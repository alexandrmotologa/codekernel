package lsp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestLSPServerStdio(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "codekernel_lsp_test_*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := graph.Open(dbPath)
	if err != nil {
		t.Fatalf("db open error: %v", err)
	}
	defer db.Close()

	// Seed test symbol
	_ = db.SaveFile(&parser.ParsedFile{
		FilePath:  "auth/jwt.go",
		Language:  "go",
		Hash:      "hash999",
		ByteSize:  200,
		LineCount: 15,
		Symbols: []parser.Symbol{
			{
				Name:          "VerifyToken",
				QualifiedName: "jwt.VerifyToken",
				Kind:          parser.KindFunction,
				Signature:     "func VerifyToken(token string) bool",
				Docstring:     "VerifyToken parses JWT claims",
				FilePath:      "auth/jwt.go",
				StartLine:     5,
				EndLine:       15,
				Exported:      true,
				Language:      "go",
				Body:          "func VerifyToken(token string) bool { return true }",
				TokenCount:    15,
			},
		},
	})
	_ = db.ResolveEdges()

	handler := NewLSPHandler(db, tempDir)

	// Prepare mock JSON-RPC request for initialize
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"processId":123,"rootPath":"` + filepath.ToSlash(tempDir) + `"}}`
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(reqBody), reqBody)

	// Prepare shutdown and exit
	exitReq := `{"jsonrpc":"2.0","id":2,"method":"shutdown"}`
	exitFrame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(exitReq), exitReq)

	finalExit := `{"jsonrpc":"2.0","method":"exit"}`
	finalExitFrame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(finalExit), finalExit)

	inBuf := bytes.NewBufferString(frame + exitFrame + finalExitFrame)
	outBuf := &bytes.Buffer{}

	server := NewServer(inBuf, outBuf, handler)
	err = server.Serve()
	if err != nil {
		t.Fatalf("server serve error: %v", err)
	}

	outStr := outBuf.String()
	if !bytes.Contains(outBuf.Bytes(), []byte(`"name":"codekernel"`)) {
		t.Fatalf("expected server name in response, got: %s", outStr)
	}

	// Test workspace/symbol directly
	syms, err := handler.HandleWorkspaceSymbol(WorkspaceSymbolParams{Query: "VerifyToken"})
	if err != nil {
		t.Fatalf("workspace symbol error: %v", err)
	}
	if len(syms) != 1 || syms[0].Name != "VerifyToken" {
		t.Fatalf("expected VerifyToken, got: %+v", syms)
	}

	// Test custom codekernel/packContext method
	packRes, err := handler.HandlePackContext(PackContextParams{Symbol: "VerifyToken", Budget: 1000})
	if err != nil {
		t.Fatalf("packContext error: %v", err)
	}
	if packRes.Symbol != "jwt.VerifyToken" {
		t.Errorf("expected jwt.VerifyToken, got %s", packRes.Symbol)
	}
}
