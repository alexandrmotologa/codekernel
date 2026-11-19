package lsp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/packer"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// LSPHandler processes individual LSP methods.
type LSPHandler struct {
	db        *graph.DB
	packer    *packer.ContextPacker
	formatter *packer.Formatter
	rootDir   string
}

// NewLSPHandler creates a handler instance.
func NewLSPHandler(db *graph.DB, rootDir string) *LSPHandler {
	return &LSPHandler{
		db:        db,
		packer:    packer.NewContextPacker(4096, true),
		formatter: packer.NewFormatter(),
		rootDir:   rootDir,
	}
}

func (h *LSPHandler) HandleInitialize(params InitializeParams) (InitializeResult, error) {
	if params.RootPath != "" {
		h.rootDir = params.RootPath
	}
	return InitializeResult{
		Capabilities: ServerCapabilities{
			DefinitionProvider:      true,
			ReferencesProvider:      true,
			WorkspaceSymbolProvider: true,
		},
		ServerInfo: ServerInfo{
			Name:    "codekernel",
			Version: "1.0.0",
		},
	}, nil
}

func (h *LSPHandler) HandleWorkspaceSymbol(params WorkspaceSymbolParams) ([]SymbolInformation, error) {
	symbols, err := h.db.FindSymbols(params.Query, 100)
	if err != nil {
		return nil, err
	}

	var results []SymbolInformation
	for _, s := range symbols {
		absPath := s.FilePath
		if !filepath.IsAbs(absPath) && h.rootDir != "" {
			absPath = filepath.Join(h.rootDir, s.FilePath)
		}

		uri := pathToURI(absPath)
		results = append(results, SymbolInformation{
			Name:          s.Name,
			Kind:          lspSymbolKind(s.Kind),
			ContainerName: s.Receiver,
			Location: Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: s.StartLine - 1, Character: 0},
					End:   Position{Line: s.EndLine - 1, Character: 0},
				},
			},
		})
	}
	return results, nil
}

func (h *LSPHandler) HandleDefinition(params TextDocumentPositionParams) ([]Location, error) {
	// Look up symbols matching the requested file
	relPath := uriToRelPath(params.TextDocument.URI, h.rootDir)
	symbols, err := h.db.FindSymbols("", 500)
	if err != nil {
		return nil, err
	}

	var matchedLocations []Location
	targetLine := params.Position.Line + 1

	for _, s := range symbols {
		if s.FilePath == relPath && s.StartLine <= targetLine && s.EndLine >= targetLine {
			// Found enclosing symbol, return its own definition
			absPath := s.FilePath
			if !filepath.IsAbs(absPath) && h.rootDir != "" {
				absPath = filepath.Join(h.rootDir, s.FilePath)
			}
			matchedLocations = append(matchedLocations, Location{
				URI: pathToURI(absPath),
				Range: Range{
					Start: Position{Line: s.StartLine - 1, Character: 0},
					End:   Position{Line: s.EndLine - 1, Character: 0},
				},
			})
			break
		}
	}

	return matchedLocations, nil
}

func (h *LSPHandler) HandleReferences(params TextDocumentPositionParams) ([]Location, error) {
	relPath := uriToRelPath(params.TextDocument.URI, h.rootDir)
	symbols, err := h.db.FindSymbols("", 500)
	if err != nil {
		return nil, err
	}

	var locations []Location
	targetLine := params.Position.Line + 1

	for _, s := range symbols {
		if s.FilePath == relPath && s.StartLine <= targetLine && s.EndLine >= targetLine {
			_, symID, err := h.db.GetSymbolByQualifiedName(s.QualifiedName)
			if err == nil && symID > 0 {
				callers, err := h.db.GetIncomingCallers(symID)
				if err == nil {
					for _, c := range callers {
						absPath := c.FilePath
						if !filepath.IsAbs(absPath) && h.rootDir != "" {
							absPath = filepath.Join(h.rootDir, c.FilePath)
						}
						locations = append(locations, Location{
							URI: pathToURI(absPath),
							Range: Range{
								Start: Position{Line: c.StartLine - 1, Character: 0},
								End:   Position{Line: c.EndLine - 1, Character: 0},
							},
						})
					}
				}
			}
			break
		}
	}

	return locations, nil
}

func (h *LSPHandler) HandlePackContext(params PackContextParams) (*PackContextResult, error) {
	budget := params.Budget
	if budget <= 0 {
		budget = 4096
	}

	// Find target symbol
	sym, symID, err := h.db.GetSymbolByQualifiedName(params.Symbol)
	if err != nil || sym == nil {
		// Try fuzzy search
		candidates, ferr := h.db.FindSymbols(params.Symbol, 1)
		if ferr != nil || len(candidates) == 0 {
			return nil, fmt.Errorf("symbol not found: %s", params.Symbol)
		}
		sym = &candidates[0]
		_, symID, _ = h.db.GetSymbolByQualifiedName(sym.QualifiedName)
	}

	subgraph, err := h.db.Neighborhood(symID, 2)
	if err != nil {
		return nil, fmt.Errorf("neighborhood failed: %w", err)
	}

	packed := h.packer.Pack(subgraph, budget)
	markdown := h.formatter.FormatMarkdown(packed)

	return &PackContextResult{
		Symbol:     sym.QualifiedName,
		Budget:     budget,
		UsedTokens: packed.UsedTokens,
		Markdown:   markdown,
		Data:       packed,
	}, nil
}

func pathToURI(absPath string) string {
	absPath = filepath.ToSlash(absPath)
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}
	u := url.URL{
		Scheme: "file",
		Path:   absPath,
	}
	return u.String()
}

func uriToRelPath(uriStr, rootDir string) string {
	u, err := url.Parse(uriStr)
	if err != nil {
		return uriStr
	}
	p := u.Path
	if len(p) > 2 && p[0] == '/' && p[2] == ':' {
		p = p[1:] // Windows drive letter
	}
	rel, err := filepath.Rel(rootDir, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

func lspSymbolKind(k parser.SymbolKind) int {
	switch k {
	case parser.KindFunction:
		return 12 // Function
	case parser.KindMethod:
		return 6 // Method
	case parser.KindClass:
		return 5 // Class
	case parser.KindInterface:
		return 11 // Interface
	case parser.KindStruct:
		return 23 // Struct
	case parser.KindEnum:
		return 10 // Enum
	case parser.KindVariable:
		return 13 // Variable
	default:
		return 12
	}
}
