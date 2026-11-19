package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/config"
	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestHTTPServerAPI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "codekernel_http_test_*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig(tempDir)
	db, err := graph.Open(cfg.Project.DatabasePath)
	if err != nil {
		t.Fatalf("db open error: %v", err)
	}
	defer db.Close()

	// Seed test symbol
	_ = db.SaveFile(&parser.ParsedFile{
		FilePath:  "auth/handler.go",
		Language:  "go",
		Hash:      "hash333",
		ByteSize:  300,
		LineCount: 20,
		Symbols: []parser.Symbol{
			{
				Name:          "HandleLogin",
				QualifiedName: "auth.HandleLogin",
				Kind:          parser.KindFunction,
				Signature:     "func HandleLogin(w http.ResponseWriter, r *http.Request)",
				Docstring:     "HandleLogin authenticates web requests",
				FilePath:      "auth/handler.go",
				StartLine:     5,
				EndLine:       25,
				Exported:      true,
				Language:      "go",
				Body:          "func HandleLogin(w http.ResponseWriter, r *http.Request) {\n    fmt.Println(\"ok\")\n}",
				TokenCount:    30,
			},
		},
	})
	_ = db.ResolveEdges()

	indexer := graph.NewIndexer(cfg, db)
	srv := NewHTTPServer(cfg, db, indexer)
	handler := srv.Handler()

	// 1. Test /api/stats
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/stats, got %d", rec.Code)
	}
	var stats graph.RepoStats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}
	if stats.TotalSymbols != 1 {
		t.Errorf("expected 1 symbol in stats, got %d", stats.TotalSymbols)
	}

	// 2. Test /api/symbols
	req = httptest.NewRequest(http.MethodGet, "/api/symbols?q=HandleLogin", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/symbols, got %d", rec.Code)
	}
	var syms []parser.Symbol
	_ = json.Unmarshal(rec.Body.Bytes(), &syms)
	if len(syms) != 1 || syms[0].Name != "HandleLogin" {
		t.Errorf("expected HandleLogin, got: %+v", syms)
	}

	// 3. Test /api/pack
	packPayload := []byte(`{"symbol":"auth.HandleLogin","budget":1000}`)
	req = httptest.NewRequest(http.MethodPost, "/api/pack", bytes.NewReader(packPayload))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/pack, got %d", rec.Code)
	}
	var packData map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &packData)
	if packData["symbol"] != "auth.HandleLogin" {
		t.Errorf("expected symbol auth.HandleLogin, got %v", packData["symbol"])
	}

	// 4. Test Embedded static index.html
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for embedded static root, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("codekernel")) {
		t.Errorf("expected codekernel title in embedded html, got: %s", rec.Body.String())
	}
}
