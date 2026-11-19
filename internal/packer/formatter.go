package packer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/codekernel/internal/chunker"
)

// Formatter serializes packed context subgraphs for consumption by agents and editors.
type Formatter struct{}

// NewFormatter creates a new formatter instance.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatMarkdown outputs agent-ready markdown context with clean provenance tags.
func (f *Formatter) FormatMarkdown(p *PackedContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Code Context Subgraph: %s\n", p.EntrySymbol))
	sb.WriteString(fmt.Sprintf("> Context Token Budget: %d / %d tokens | Compressed: %d | Omitted: %d\n\n",
		p.UsedTokens, p.Budget, p.CompressedCount, p.DroppedCount))

	// 1. Target symbol
	sb.WriteString("### Target Symbol Definition\n")
	sb.WriteString(f.formatChunkBlock(p.TargetChunk))
	sb.WriteString("\n\n")

	// 2. Direct callers
	if len(p.CallerChunks) > 0 {
		sb.WriteString(fmt.Sprintf("### Direct Callers (%d)\n", len(p.CallerChunks)))
		for _, c := range p.CallerChunks {
			sb.WriteString(f.formatChunkBlock(c))
			sb.WriteString("\n\n")
		}
	}

	// 3. Immediate Callees
	if len(p.CalleeChunks) > 0 {
		sb.WriteString(fmt.Sprintf("### Immediate Dependencies (%d)\n", len(p.CalleeChunks)))
		for _, c := range p.CalleeChunks {
			sb.WriteString(f.formatChunkBlock(c))
			sb.WriteString("\n\n")
		}
	}

	// 4. Other neighborhood symbols
	if len(p.OtherChunks) > 0 {
		sb.WriteString(fmt.Sprintf("### Related Neighborhood Symbols (%d)\n", len(p.OtherChunks)))
		for _, c := range p.OtherChunks {
			sb.WriteString(f.formatChunkBlock(c))
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

func (f *Formatter) formatChunkBlock(c chunker.Chunk) string {
	ext := "text"
	if strings.HasSuffix(c.FilePath, ".go") {
		ext = "go"
	} else if strings.HasSuffix(c.FilePath, ".ts") || strings.HasSuffix(c.FilePath, ".tsx") {
		ext = "typescript"
	} else if strings.HasSuffix(c.FilePath, ".js") {
		ext = "javascript"
	} else if strings.HasSuffix(c.FilePath, ".py") {
		ext = "python"
	} else if strings.HasSuffix(c.FilePath, ".rs") {
		ext = "rust"
	} else if strings.HasSuffix(c.FilePath, ".java") {
		ext = "java"
	}

	header := fmt.Sprintf("// File: %s (lines %d-%d) [%s: %s | ~%d tokens]",
		c.FilePath, c.StartLine, c.EndLine, c.Kind, c.SymbolName, c.TokenCount)

	return fmt.Sprintf("%s\n```%s\n%s\n```", header, ext, c.Content)
}

// FormatJSON outputs the packed context as serialized JSON.
func (f *Formatter) FormatJSON(p *PackedContext) ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}
