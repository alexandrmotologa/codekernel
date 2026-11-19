package parser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// LanguageParser defines language-specific AST extractor behavior.
type LanguageParser interface {
	Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error)
}

// Engine manages tree-sitter parsers and dispatches source files to language extractors.
type Engine struct {
	parsers   map[string]*sync.Pool
	extractors map[string]LanguageParser
}

// NewEngine initializes language grammars and parser pools.
func NewEngine() *Engine {
	engine := &Engine{
		parsers:    make(map[string]*sync.Pool),
		extractors: make(map[string]LanguageParser),
	}

	// Register Go
	engine.registerLanguage("go", golang.GetLanguage(), &GoExtractor{})

	// Register TypeScript and TSX
	engine.registerLanguage("typescript", typescript.GetLanguage(), &TypeScriptExtractor{isTSX: false})
	engine.registerLanguage("tsx", tsx.GetLanguage(), &TypeScriptExtractor{isTSX: true})
	engine.registerLanguage("javascript", javascript.GetLanguage(), &TypeScriptExtractor{isTSX: false})

	// Register Python
	engine.registerLanguage("python", python.GetLanguage(), &PythonExtractor{})

	// Register Rust
	engine.registerLanguage("rust", rust.GetLanguage(), &RustExtractor{})

	// Register Java
	engine.registerLanguage("java", java.GetLanguage(), &JavaExtractor{})

	return engine
}

func (e *Engine) registerLanguage(lang string, grammar *sitter.Language, extractor LanguageParser) {
	e.parsers[lang] = &sync.Pool{
		New: func() any {
			p := sitter.NewParser()
			p.SetLanguage(grammar)
			return p
		},
	}
	e.extractors[lang] = extractor
}

// DetectLanguage maps file extension to supported language identifier.
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".mts", ".cts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".js", ".mjs", ".cjs", ".jsx":
		return "javascript"
	case ".py", ".pyw":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	default:
		return ""
	}
}

// EstimateTokens provides a deterministic token count estimate based on code tokens.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	tokens := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			tokens++
			inWord = false
			continue
		}
		if !inWord {
			tokens++
			inWord = true
		}
	}

	if tokens == 0 && len(text) > 0 {
		tokens = (len(text) + 3) / 4
	}
	return tokens
}

// Parse extracts all symbols, calls, and imports from file content.
func (e *Engine) Parse(filePath string, content []byte) (*ParsedFile, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}

	pool, ok := e.parsers[lang]
	if !ok {
		return nil, fmt.Errorf("parser not registered for language: %s", lang)
	}

	extractor, ok := e.extractors[lang]
	if !ok {
		return nil, fmt.Errorf("extractor not registered for language: %s", lang)
	}

	p := pool.Get().(*sitter.Parser)
	defer pool.Put(p)

	tree, err := p.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("parse tree error: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	symbols, calls, imports, err := extractor.Parse(root, content, filePath)
	if err != nil {
		return nil, fmt.Errorf("extraction error: %w", err)
	}

	// Calculate line count and SHA256 hash
	lines := strings.Count(string(content), "\n") + 1
	hasher := sha256.New()
	hasher.Write(content)
	hashStr := hex.EncodeToString(hasher.Sum(nil))

	return &ParsedFile{
		FilePath:  filePath,
		Language:  lang,
		Content:   content,
		Symbols:   symbols,
		Calls:     calls,
		Imports:   imports,
		LineCount: lines,
		ByteSize:  len(content),
		Hash:      hashStr,
	}, nil
}
