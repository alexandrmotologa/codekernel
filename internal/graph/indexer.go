package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/codekernel/internal/config"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// IndexProgress captures real-time indexing metrics for listeners.
type IndexProgress struct {
	TotalFiles     int           `json:"total_files"`
	ProcessedFiles int           `json:"processed_files"`
	SkippedFiles   int           `json:"skipped_files"`
	IndexedFiles   int           `json:"indexed_files"`
	TotalSymbols   int           `json:"total_symbols"`
	CurrentFile    string        `json:"current_file"`
	Elapsed        time.Duration `json:"elapsed"`
	Completed      bool          `json:"completed"`
	Error          string        `json:"error,omitempty"`
}

// Indexer orchestrates file discovery, incremental hashing, and parallel AST extraction.
type Indexer struct {
	cfg       *config.Config
	db        *DB
	engine    *parser.Engine
	listeners []chan IndexProgress
	mu        sync.Mutex
}

// NewIndexer creates an indexer instance.
func NewIndexer(cfg *config.Config, db *DB) *Indexer {
	return &Indexer{
		cfg:    cfg,
		db:     db,
		engine: parser.NewEngine(),
	}
}

// Subscribe registers a channel to receive live indexing progress updates.
func (idx *Indexer) Subscribe() chan IndexProgress {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	ch := make(chan IndexProgress, 50)
	idx.listeners = append(idx.listeners, ch)
	return ch
}

func (idx *Indexer) emit(p IndexProgress) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for _, ch := range idx.listeners {
		select {
		case ch <- p:
		default:
		}
	}
}

// IndexRepository scans the target directory and updates the relational index.
func (idx *Indexer) IndexRepository() (*IndexProgress, error) {
	startTime := time.Now()
	rootDir := idx.cfg.Project.RootDir

	// 1. Discover all candidate source files
	var candidateFiles []string
	maxSize := int64(idx.cfg.Parser.MaxFileSizeKB) * 1024
	if maxSize <= 0 {
		maxSize = 1024 * 1024
	}

	excludeDirs := make(map[string]bool)
	for _, d := range idx.cfg.Parser.ExcludeDirs {
		excludeDirs[d] = true
	}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		if relPath == "." {
			return nil
		}

		if d.IsDir() {
			base := filepath.Base(path)
			if excludeDirs[base] || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check supported language extension
		lang := parser.DetectLanguage(path)
		if lang == "" {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > maxSize {
			return nil
		}

		candidateFiles = append(candidateFiles, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	progress := IndexProgress{
		TotalFiles: len(candidateFiles),
	}
	idx.emit(progress)

	// 2. Process files with incremental hashing
	type parseJob struct {
		absPath string
		relPath string
		content []byte
		hash    string
	}

	jobs := make(chan parseJob, 100)
	results := make(chan *parser.ParsedFile, 100)
	var wg sync.WaitGroup

	numWorkers := 4
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				pf, err := idx.engine.Parse(job.relPath, job.content)
				if err == nil && pf != nil {
					pf.Hash = job.hash
					results <- pf
				} else {
					results <- nil
				}
			}
		}()
	}

	// Collector goroutine to save to SQLite
	var saveWg sync.WaitGroup
	saveWg.Add(1)
	totalSymbols := 0
	go func() {
		defer saveWg.Done()
		for pf := range results {
			if pf != nil {
				_ = idx.db.SaveFile(pf)
				totalSymbols += len(pf.Symbols)
			}
		}
	}()

	// Feed candidate files
	for _, absPath := range candidateFiles {
		relPath, _ := filepath.Rel(rootDir, absPath)
		// normalize to forward slashes for cross-platform stability
		relPath = filepath.ToSlash(relPath)

		content, err := os.ReadFile(absPath)
		if err != nil {
			progress.SkippedFiles++
			continue
		}

		// Compute hash
		hasher := sha256.New()
		hasher.Write(content)
		hashStr := hex.EncodeToString(hasher.Sum(nil))

		// Check if file exists in db with identical hash
		rec, _ := idx.db.GetFileByPath(relPath)
		if rec != nil && rec.Hash == hashStr {
			progress.SkippedFiles++
			progress.ProcessedFiles++
			continue
		}

		progress.IndexedFiles++
		progress.ProcessedFiles++
		progress.CurrentFile = relPath
		progress.Elapsed = time.Since(startTime)
		idx.emit(progress)

		jobs <- parseJob{
			absPath: absPath,
			relPath: relPath,
			content: content,
			hash:    hashStr,
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	saveWg.Wait()

	// 3. Resolve caller/callee cross-file symbol edges
	_ = idx.db.ResolveEdges()

	progress.TotalSymbols = totalSymbols
	progress.Elapsed = time.Since(startTime)
	progress.Completed = true
	idx.emit(progress)

	return &progress, nil
}

// ComputeHash computes a SHA256 string for arbitrary reader.
func ComputeHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
