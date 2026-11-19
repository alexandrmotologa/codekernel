package chunker

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// Chunk represents an AST-aligned semantic segment of code.
type Chunk struct {
	FilePath   string            `json:"file_path"`
	SymbolName string            `json:"symbol_name"`
	Kind       parser.SymbolKind `json:"kind"`
	Content    string            `json:"content"`
	StartLine  int               `json:"start_line"`
	EndLine    int               `json:"end_line"`
	TokenCount int               `json:"token_count"`
}

// Chunker segments source files based on syntactic AST boundaries.
type Chunker struct{}

// NewChunker creates a new AST chunker.
func NewChunker() *Chunker {
	return &Chunker{}
}

// ChunkFile partitions a parsed file into semantic units without slicing function bodies.
func (c *Chunker) ChunkFile(pf *parser.ParsedFile) []Chunk {
	if pf == nil || len(pf.Content) == 0 {
		return nil
	}

	lines := strings.Split(string(pf.Content), "\n")
	var chunks []Chunk

	// If no symbols found, return file as single chunk
	if len(pf.Symbols) == 0 {
		return []Chunk{
			{
				FilePath:   pf.FilePath,
				SymbolName: "<file>",
				Kind:       parser.KindModule,
				Content:    string(pf.Content),
				StartLine:  1,
				EndLine:    len(lines),
				TokenCount: parser.EstimateTokens(string(pf.Content)),
			},
		}
	}

	// 1. Module prelude (lines before first symbol, such as package clause and imports)
	firstSymStart := pf.Symbols[0].StartLine
	if firstSymStart > 1 {
		preludeLines := lines[:firstSymStart-1]
		preludeText := strings.TrimSpace(strings.Join(preludeLines, "\n"))
		if len(preludeText) > 0 {
			chunks = append(chunks, Chunk{
				FilePath:   pf.FilePath,
				SymbolName: "<prelude>",
				Kind:       parser.KindModule,
				Content:    preludeText,
				StartLine:  1,
				EndLine:    firstSymStart - 1,
				TokenCount: parser.EstimateTokens(preludeText),
			})
		}
	}

	// 2. Each top-level symbol represents an atomic, uncut chunk
	for _, sym := range pf.Symbols {
		content := sym.Body
		if sym.Docstring != "" {
			content = sym.Docstring + "\n" + content
		}

		chunks = append(chunks, Chunk{
			FilePath:   pf.FilePath,
			SymbolName: sym.QualifiedName,
			Kind:       sym.Kind,
			Content:    content,
			StartLine:  sym.StartLine,
			EndLine:    sym.EndLine,
			TokenCount: parser.EstimateTokens(content),
		})
	}

	return chunks
}

// FormatContextChunk prepares a chunk for agent prompt injection with source path tags.
func FormatContextChunk(chunk Chunk) string {
	header := fmt.Sprintf("// Location: %s (lines %d-%d)", chunk.FilePath, chunk.StartLine, chunk.EndLine)
	if chunk.SymbolName != "" && chunk.SymbolName != "<prelude>" {
		header += fmt.Sprintf(" [%s: %s]", chunk.Kind, chunk.SymbolName)
	}
	return header + "\n" + chunk.Content
}
