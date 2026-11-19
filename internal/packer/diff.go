package packer

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// ModifiedSpan represents a changed region in a file.
type ModifiedSpan struct {
	FilePath  string
	StartLine int
	LineCount int
}

// DiffPacker gathers context around all git-modified symbols.
type DiffPacker struct {
	db      *graph.DB
	packer  *ContextPacker
	rootDir string
}

// NewDiffPacker creates a diff packer instance.
func NewDiffPacker(db *graph.DB, rootDir string) *DiffPacker {
	return &DiffPacker{
		db:      db,
		packer:  NewContextPacker(4096, true),
		rootDir: rootDir,
	}
}

// GetChangedSpans runs git diff to determine modified line ranges.
func (dp *DiffPacker) GetChangedSpans(gitRef string) ([]ModifiedSpan, error) {
	args := []string{"diff", "-U0"}
	if gitRef != "" {
		args = append(args, gitRef)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dp.rootDir
	out, err := cmd.Output()
	if err != nil {
		// Fallback to checking unstaged changes via diff HEAD or status
		cmd = exec.Command("git", "diff", "HEAD", "-U0")
		cmd.Dir = dp.rootDir
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff failed: %w", err)
		}
	}

	var spans []ModifiedSpan
	scanner := bufio.NewScanner(bytes.NewReader(out))
	currentFile := ""

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = filepath.ToSlash(strings.TrimPrefix(line, "+++ b/"))
			continue
		}

		// Parse hunk header: @@ -a,b +c,d @@ or @@ -a +c @@
		if strings.HasPrefix(line, "@@ ") && currentFile != "" {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				newRange := parts[2] // +c,d or +c
				if strings.HasPrefix(newRange, "+") {
					newRange = strings.TrimPrefix(newRange, "+")
					subParts := strings.Split(newRange, ",")
					startLine, _ := strconv.Atoi(subParts[0])
					count := 1
					if len(subParts) > 1 {
						count, _ = strconv.Atoi(subParts[1])
					}
					spans = append(spans, ModifiedSpan{
						FilePath:  currentFile,
						StartLine: startLine,
						LineCount: count,
					})
				}
			}
		}
	}

	return spans, nil
}

// PackChangedSymbols packs all symbols touched by recent modifications into a unified prompt context.
func (dp *DiffPacker) PackChangedSymbols(gitRef string, budget int) (string, error) {
	spans, err := dp.GetChangedSpans(gitRef)
	if err != nil {
		return "", err
	}

	if len(spans) == 0 {
		return "No modified symbols detected in working directory.", nil
	}

	// Match spans to indexed symbols
	allSymbols, err := dp.db.FindSymbols("", 2000)
	if err != nil {
		return "", err
	}

	matchedSymbols := make(map[string]parser.Symbol)
	for _, span := range spans {
		spanEnd := span.StartLine + span.LineCount
		for _, sym := range allSymbols {
			if sym.FilePath == span.FilePath {
				// Check overlap
				if (sym.StartLine <= spanEnd) && (sym.EndLine >= span.StartLine) {
					matchedSymbols[sym.QualifiedName] = sym
				}
			}
		}
	}

	if len(matchedSymbols) == 0 {
		return "Modifications do not overlap with indexed AST symbols.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Diff-Aware Context: %d Modified Symbols\n\n", len(matchedSymbols)))

	perSymbolBudget := budget / len(matchedSymbols)
	if perSymbolBudget < 500 {
		perSymbolBudget = 500
	}

	formatter := NewFormatter()
	for _, sym := range matchedSymbols {
		_, symID, err := dp.db.GetSymbolByQualifiedName(sym.QualifiedName)
		if err == nil && symID > 0 {
			subgraph, err := dp.db.Neighborhood(symID, 1)
			if err == nil {
				packed := dp.packer.Pack(subgraph, perSymbolBudget)
				sb.WriteString(formatter.FormatMarkdown(packed))
				sb.WriteString("\n\n---\n\n")
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}
