package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestMCPServerStdio(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "codekernel_mcp_test_*")
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

	// Seed symbol
	_ = db.SaveFile(&parser.ParsedFile{
		FilePath:  "auth/token.go",
		Language:  "go",
		Hash:      "hash444",
		ByteSize:  250,
		LineCount: 20,
		Symbols: []parser.Symbol{
			{
				Name:          "GenerateToken",
				QualifiedName: "auth.GenerateToken",
				Kind:          parser.KindFunction,
				Signature:     "func GenerateToken(userID string) string",
				Docstring:     "GenerateToken produces a signed JWT",
				FilePath:      "auth/token.go",
				StartLine:     5,
				EndLine:       15,
				Exported:      true,
				Language:      "go",
				Body:          "func GenerateToken(userID string) string { return \"jwt.xyz\" }",
				TokenCount:    20,
			},
		},
	})
	_ = db.ResolveEdges()

	// Prepare mock JSON-RPC lines
	req1 := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	req2 := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	req3 := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_symbols","arguments":{"query":"GenerateToken"}}}`
	req4 := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"pack_context","arguments":{"symbol":"auth.GenerateToken","budget":2000}}}`

	input := strings.Join([]string{req1, req2, req3, req4}, "\n") + "\n"
	inBuf := bytes.NewBufferString(input)
	outBuf := &bytes.Buffer{}

	server := NewServer(inBuf, outBuf, db, tempDir)
	err = server.Serve()
	if err != nil {
		t.Fatalf("mcp server serve error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected 4 responses, got %d: %s", len(lines), outBuf.String())
	}

	// Verify initialize response
	var initResp JSONRPCMessage
	_ = json.Unmarshal([]byte(lines[0]), &initResp)
	if initResp.Result == nil {
		t.Errorf("empty init result")
	}

	// Verify tools list
	var listResp JSONRPCMessage
	_ = json.Unmarshal([]byte(lines[1]), &listResp)
	listBytes, _ := json.Marshal(listResp.Result)
	var listResult ListToolsResult
	_ = json.Unmarshal(listBytes, &listResult)
	if len(listResult.Tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(listResult.Tools))
	}

	// Verify pack_context
	var packResp JSONRPCMessage
	_ = json.Unmarshal([]byte(lines[3]), &packResp)
	packBytes, _ := json.Marshal(packResp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(packBytes, &callResult)
	if len(callResult.Content) == 0 || !strings.Contains(callResult.Content[0].Text, "GenerateToken") {
		t.Errorf("unexpected pack tool result: %+v", callResult)
	}
}
