package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/packer"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// Server implements Model Context Protocol over stdio streams.
type Server struct {
	reader    *bufio.Reader
	writer    io.Writer
	db        *graph.DB
	packer    *packer.ContextPacker
	formatter *packer.Formatter
	rootDir   string
	mu        sync.Mutex
	running   bool
}

// NewServer creates a new MCP server.
func NewServer(r io.Reader, w io.Writer, db *graph.DB, rootDir string) *Server {
	return &Server{
		reader:    bufio.NewReader(r),
		writer:    w,
		db:        db,
		packer:    packer.NewContextPacker(4096, true),
		formatter: packer.NewFormatter(),
		rootDir:   rootDir,
		running:   true,
	}
}

// Serve starts reading JSON-RPC lines from the client.
func (s *Server) Serve() error {
	for s.running {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		s.dispatch(&msg)
	}
	return nil
}

func (s *Server) writeResponse(msg *JSONRPCMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_, _ = s.writer.Write(body)
	_, _ = s.writer.Write([]byte("\n"))
}

func (s *Server) dispatch(msg *JSONRPCMessage) {
	resp := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}

	switch msg.Method {
	case "initialize":
		resp.Result = InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: map[string]any{},
			},
			ServerInfo: ServerInfo{
				Name:    "codekernel",
				Version: "1.0.0",
			},
		}

	case "notifications/initialized":
		return // Notification, no reply

	case "tools/list":
		resp.Result = ListToolsResult{
			Tools: s.getToolDefinitions(),
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			resp.Error = &ResponseError{Code: -32602, Message: "Invalid tool params: " + err.Error()}
		} else {
			resp.Result = s.handleToolCall(params)
		}

	default:
		if msg.ID != nil {
			resp.Error = &ResponseError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", msg.Method)}
		} else {
			return
		}
	}

	if msg.ID != nil {
		s.writeResponse(resp)
	}
}

func (s *Server) getToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "search_symbols",
			Description: "Search indexed codebase symbols, types, and functions using FTS5 and BM25 ranking.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query keyword, qualified name, or docstring concept"},
					"limit": map[string]any{"type": "integer", "description": "Maximum number of symbols to return (default 20)"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "pack_context",
			Description: "Extract AST-aware surgical prompt context subgraph for a target symbol within an exact token budget.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol": map[string]any{"type": "string", "description": "Target symbol name or qualified name"},
					"budget": map[string]any{"type": "integer", "description": "Maximum context token budget (default 4096)"},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "get_call_graph",
			Description: "Extract call graph neighborhood (incoming callers and outgoing dependencies) for a symbol.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symbol": map[string]any{"type": "string", "description": "Target symbol qualified name"},
					"hops":   map[string]any{"type": "integer", "description": "Neighborhood depth (1-3, default 2)"},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "get_file_outline",
			Description: "Retrieve AST outline of symbols, classes, and methods declared in a source file.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative file path (e.g. internal/parser/treesitter.go)"},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (s *Server) handleToolCall(params CallToolParams) CallToolResult {
	switch params.Name {
	case "search_symbols":
		var args struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if args.Limit <= 0 {
			args.Limit = 20
		}
		symbols, err := s.db.FindSymbols(args.Query, args.Limit)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: err.Error()}}}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d symbols for '%s':\n\n", len(symbols), args.Query))
		for _, sym := range symbols {
			sb.WriteString(fmt.Sprintf("• %s [%s, %s] (lines %d-%d in %s)\n  Signature: %s\n\n",
				sym.QualifiedName, sym.Kind, sym.Language, sym.StartLine, sym.EndLine, sym.FilePath, sym.Signature))
		}
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: strings.TrimSpace(sb.String())}}}

	case "pack_context":
		var args struct {
			Symbol string `json:"symbol"`
			Budget int    `json:"budget"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if args.Budget <= 0 {
			args.Budget = 4096
		}

		sym, symID, err := s.db.GetSymbolByQualifiedName(args.Symbol)
		if err != nil || sym == nil {
			cands, _ := s.db.FindSymbols(args.Symbol, 1)
			if len(cands) == 0 {
				return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Symbol not found: " + args.Symbol}}}
			}
			sym = &cands[0]
			_, symID, _ = s.db.GetSymbolByQualifiedName(sym.QualifiedName)
		}

		subgraph, err := s.db.Neighborhood(symID, 2)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: err.Error()}}}
		}

		packed := s.packer.Pack(subgraph, args.Budget)
		md := s.formatter.FormatMarkdown(packed)
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: md}}}

	case "get_call_graph":
		var args struct {
			Symbol string `json:"symbol"`
			Hops   int    `json:"hops"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		if args.Hops <= 0 {
			args.Hops = 2
		}

		sym, symID, err := s.db.GetSymbolByQualifiedName(args.Symbol)
		if err != nil || sym == nil {
			cands, _ := s.db.FindSymbols(args.Symbol, 1)
			if len(cands) == 0 {
				return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Symbol not found: " + args.Symbol}}}
			}
			sym = &cands[0]
			_, symID, _ = s.db.GetSymbolByQualifiedName(sym.QualifiedName)
		}

		subgraph, err := s.db.Neighborhood(symID, args.Hops)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: err.Error()}}}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Call Graph for %s (%d nodes, %d edges):\n\n", sym.QualifiedName, len(subgraph.Nodes), len(subgraph.Edges)))
		for _, n := range subgraph.Nodes {
			sb.WriteString(fmt.Sprintf("- [%s] %s (Distance: %d, Weight: %.2f) in %s\n",
				n.Role, n.Symbol.QualifiedName, n.Distance, n.Weight, n.Symbol.FilePath))
		}
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: strings.TrimSpace(sb.String())}}}

	case "get_file_outline":
		var args struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(params.Arguments, &args)
		normPath := filepath.ToSlash(args.Path)

		absPath := filepath.Join(s.rootDir, normPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "File read error: " + err.Error()}}}
		}

		engine := parser.NewEngine()
		pf, err := engine.Parse(normPath, content)
		if err != nil {
			return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Parse error: " + err.Error()}}}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("File Outline: %s (%s, %d lines, %d symbols)\n\n", normPath, pf.Language, pf.LineCount, len(pf.Symbols)))
		for _, s := range pf.Symbols {
			sb.WriteString(fmt.Sprintf("• [%s] %s (lines %d-%d)\n  %s\n", s.Kind, s.Name, s.StartLine, s.EndLine, s.Signature))
		}
		return CallToolResult{Content: []ToolContent{{Type: "text", Text: strings.TrimSpace(sb.String())}}}

	default:
		return CallToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "Unknown tool: " + params.Name}}}
	}
}
