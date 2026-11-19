package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alexandrmotologa/codekernel/internal/config"
	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/packer"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// HTTPServer handles REST endpoints and serves the embedded React 19 Graph Studio.
type HTTPServer struct {
	cfg       *config.Config
	db        *graph.DB
	indexer   *graph.Indexer
	packer    *packer.ContextPacker
	formatter *packer.Formatter
	mux       *http.ServeMux
}

// NewHTTPServer initializes routes and dependencies.
func NewHTTPServer(cfg *config.Config, db *graph.DB, indexer *graph.Indexer) *HTTPServer {
	s := &HTTPServer{
		cfg:       cfg,
		db:        db,
		indexer:   indexer,
		packer:    packer.NewContextPacker(cfg.Packing.DefaultBudget, cfg.Packing.CompressCallees),
		formatter: packer.NewFormatter(),
		mux:       http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *HTTPServer) registerRoutes() {
	s.mux.HandleFunc("/api/stats", s.handleStats)
	s.mux.HandleFunc("/api/files", s.handleFiles)
	s.mux.HandleFunc("/api/file", s.handleFileDetail)
	s.mux.HandleFunc("/api/symbols", s.handleSymbols)
	s.mux.HandleFunc("/api/graph", s.handleGraph)
	s.mux.HandleFunc("/api/neighborhood", s.handleNeighborhood)
	s.mux.HandleFunc("/api/pack", s.handlePack)
	s.mux.HandleFunc("/api/index", s.handleTriggerIndex)
	s.mux.HandleFunc("/api/events", s.handleEvents)

	// Embedded static SPA
	s.mux.HandleFunc("/", s.handleStatic)
}

// Handler returns the underlying http.Handler with CORS enabled.
func (s *HTTPServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

// Start runs the HTTP listener on configured host:port.
func (s *HTTPServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, stats)
}

func (s *HTTPServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	files, err := s.db.ListFiles()
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []graph.FileRecord{}
	}
	s.jsonResponse(w, files)
}

func (s *HTTPServer) handleFileDetail(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		s.jsonError(w, "missing path query param", http.StatusBadRequest)
		return
	}

	absPath := filepath.Join(s.cfg.Project.RootDir, relPath)
	content, err := os.ReadFile(absPath)
	if err != nil {
		s.jsonError(w, "failed to read file: "+err.Error(), http.StatusNotFound)
		return
	}

	// Fetch symbols belonging to this file
	symbols, err := s.db.FindSymbols("", 1000)
	var fileSymbols []parser.Symbol
	if err == nil {
		for _, sym := range symbols {
			if sym.FilePath == relPath {
				fileSymbols = append(fileSymbols, sym)
			}
		}
	}

	s.jsonResponse(w, map[string]any{
		"path":     relPath,
		"content":  string(content),
		"symbols":  fileSymbols,
		"language": parser.DetectLanguage(relPath),
	})
}

func (s *HTTPServer) handleSymbols(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	symbols, err := s.db.FindSymbols(q, limit)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if symbols == nil {
		symbols = []parser.Symbol{}
	}
	s.jsonResponse(w, symbols)
}

// GraphNode format for 2D/3D force visualizer.
type GraphNode struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Qualified string `json:"qualified"`
	Kind      string `json:"kind"`
	Language  string `json:"language"`
	FilePath  string `json:"file_path"`
	Exported  bool   `json:"exported"`
	Tokens    int    `json:"tokens"`
}

// GraphEdge format for force visualizer.
type GraphEdge struct {
	Source int64  `json:"source"`
	Target int64  `json:"target"`
	Type   string `json:"type"`
}

func (s *HTTPServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 500
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	edges, err := s.db.GetAllEdges(limit)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect unique symbols involved
	nodeMap := make(map[int64]GraphNode)
	graphEdges := make([]GraphEdge, 0)

	for _, e := range edges {
		if e.CallerSymbolID != nil && e.CalleeSymbolID != nil {
			graphEdges = append(graphEdges, GraphEdge{
				Source: *e.CallerSymbolID,
				Target: *e.CalleeSymbolID,
				Type:   e.EdgeType,
			})

			// Populate nodes if not already loaded
			for _, id := range []int64{*e.CallerSymbolID, *e.CalleeSymbolID} {
				if _, ok := nodeMap[id]; !ok {
					sym, _, err := s.db.GetSymbolByQualifiedName("")
					_ = err
					_ = sym
				}
			}
		}
	}

	// Fetch all symbols to build node dictionary
	allSymbols, err := s.db.FindSymbols("", limit)
	graphNodes := make([]GraphNode, 0)
	if err == nil {
		for _, sym := range allSymbols {
			_, sid, _ := s.db.GetSymbolByQualifiedName(sym.QualifiedName)
			graphNodes = append(graphNodes, GraphNode{
				ID:        sid,
				Name:      sym.Name,
				Qualified: sym.QualifiedName,
				Kind:      string(sym.Kind),
				Language:  sym.Language,
				FilePath:  sym.FilePath,
				Exported:  sym.Exported,
				Tokens:    sym.TokenCount,
			})
		}
	}

	s.jsonResponse(w, map[string]any{
		"nodes": graphNodes,
		"edges": graphEdges,
	})
}

func (s *HTTPServer) handleNeighborhood(w http.ResponseWriter, r *http.Request) {
	symbolQuery := r.URL.Query().Get("symbol")
	if symbolQuery == "" {
		s.jsonError(w, "missing symbol parameter", http.StatusBadRequest)
		return
	}

	hopsStr := r.URL.Query().Get("hops")
	hops := 2
	if hopsStr != "" {
		if h, err := strconv.Atoi(hopsStr); err == nil && h > 0 {
			hops = h
		}
	}

	sym, symID, err := s.db.GetSymbolByQualifiedName(symbolQuery)
	if err != nil || sym == nil {
		// Fuzzy fallback
		candidates, _ := s.db.FindSymbols(symbolQuery, 1)
		if len(candidates) == 0 {
			s.jsonError(w, "symbol not found: "+symbolQuery, http.StatusNotFound)
			return
		}
		sym = &candidates[0]
		_, symID, _ = s.db.GetSymbolByQualifiedName(sym.QualifiedName)
	}

	subgraph, err := s.db.Neighborhood(symID, hops)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, subgraph)
}

type PackRequest struct {
	Symbol string `json:"symbol"`
	Budget int    `json:"budget"`
}

func (s *HTTPServer) handlePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid json body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Budget <= 0 {
		req.Budget = s.cfg.Packing.DefaultBudget
	}

	sym, symID, err := s.db.GetSymbolByQualifiedName(req.Symbol)
	if err != nil || sym == nil {
		candidates, _ := s.db.FindSymbols(req.Symbol, 1)
		if len(candidates) == 0 {
			s.jsonError(w, "symbol not found: "+req.Symbol, http.StatusNotFound)
			return
		}
		sym = &candidates[0]
		_, symID, _ = s.db.GetSymbolByQualifiedName(sym.QualifiedName)
	}

	subgraph, err := s.db.Neighborhood(symID, s.cfg.Packing.MaxHops)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	packed := s.packer.Pack(subgraph, req.Budget)
	markdown := s.formatter.FormatMarkdown(packed)

	s.jsonResponse(w, map[string]any{
		"symbol":      sym.QualifiedName,
		"budget":      req.Budget,
		"used_tokens": packed.UsedTokens,
		"markdown":    markdown,
		"packed":      packed,
	})
}

func (s *HTTPServer) handleTriggerIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		_, _ = s.indexer.IndexRepository()
	}()

	s.jsonResponse(w, map[string]string{
		"status": "indexing_started",
	})
}

func (s *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := s.indexer.Subscribe()
	defer func() {
		// channel cleanup
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case prog := <-ch:
			data, err := json.Marshal(prog)
			if err == nil {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
			if prog.Completed {
				return
			}
		}
	}
}

func (s *HTTPServer) jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
