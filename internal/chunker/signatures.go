package chunker

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// CompressSignature turns a symbol declaration into a typed signature stub without full body.
func CompressSignature(sym parser.Symbol) Chunk {
	var stubContent string
	linesCount := sym.EndLine - sym.StartLine + 1

	switch sym.Kind {
	case parser.KindFunction, parser.KindMethod:
		docPrefix := ""
		if sym.Docstring != "" {
			docPrefix = sym.Docstring + "\n"
		}
		stubComment := fmt.Sprintf(" { /* body omitted (%d lines) */ }", linesCount)
		stubContent = docPrefix + sym.Signature + stubComment

	case parser.KindClass, parser.KindStruct, parser.KindInterface:
		docPrefix := ""
		if sym.Docstring != "" {
			docPrefix = sym.Docstring + "\n"
		}
		// For structs and interfaces, summarize declaration header
		stubComment := fmt.Sprintf(" { /* definition omitted (%d lines) */ }", linesCount)
		stubContent = docPrefix + sym.Signature + stubComment

	default:
		stubContent = sym.Signature
	}

	return Chunk{
		FilePath:   sym.FilePath,
		SymbolName: sym.QualifiedName,
		Kind:       sym.Kind,
		Content:    stubContent,
		StartLine:  sym.StartLine,
		EndLine:    sym.EndLine,
		TokenCount: parser.EstimateTokens(stubContent),
	}
}

// CompressSymbolBody returns full symbol chunk if within budget, or compressed signature if required.
func CompressSymbolBody(sym parser.Symbol, mustCompress bool) Chunk {
	if mustCompress {
		return CompressSignature(sym)
	}

	content := sym.Body
	if sym.Docstring != "" && !strings.Contains(content, sym.Docstring) {
		content = sym.Docstring + "\n" + content
	}

	return Chunk{
		FilePath:   sym.FilePath,
		SymbolName: sym.QualifiedName,
		Kind:       sym.Kind,
		Content:    content,
		StartLine:  sym.StartLine,
		EndLine:    sym.EndLine,
		TokenCount: parser.EstimateTokens(content),
	}
}
