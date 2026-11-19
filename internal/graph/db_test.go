package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestGraphDBAndTraversal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "codekernel_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// 1. Create mock parsed files representing caller and callee
	file1 := &parser.ParsedFile{
		FilePath:  "service/payment.go",
		Language:  "go",
		Hash:      "hash123",
		ByteSize:  500,
		LineCount: 30,
		Symbols: []parser.Symbol{
			{
				Name:          "ProcessPayment",
				QualifiedName: "payment.ProcessPayment",
				Kind:          parser.KindFunction,
				Signature:     "func ProcessPayment(amount int) bool",
				Docstring:     "ProcessPayment handles charge authorization",
				FilePath:      "service/payment.go",
				StartLine:     10,
				EndLine:       25,
				Exported:      true,
				Language:      "go",
				Body:          "func ProcessPayment(amount int) bool {\n    return Authorize(amount)\n}",
				TokenCount:    25,
			},
		},
		Calls: []parser.CallReference{
			{
				CallerName: "ProcessPayment",
				CalleeName: "Authorize",
				FilePath:   "service/payment.go",
				Line:       15,
			},
		},
	}

	file2 := &parser.ParsedFile{
		FilePath:  "gateway/stripe.go",
		Language:  "go",
		Hash:      "hash456",
		ByteSize:  400,
		LineCount: 25,
		Symbols: []parser.Symbol{
			{
				Name:          "Authorize",
				QualifiedName: "stripe.Authorize",
				Kind:          parser.KindFunction,
				Signature:     "func Authorize(amount int) bool",
				Docstring:     "Authorize calls payment gateway",
				FilePath:      "gateway/stripe.go",
				StartLine:     5,
				EndLine:       20,
				Exported:      true,
				Language:      "go",
				Body:          "func Authorize(amount int) bool {\n    return true\n}",
				TokenCount:    20,
			},
		},
	}

	// 2. Save both files
	if err := db.SaveFile(file1); err != nil {
		t.Fatalf("failed to save file1: %v", err)
	}
	if err := db.SaveFile(file2); err != nil {
		t.Fatalf("failed to save file2: %v", err)
	}

	// 3. Resolve edges
	if err := db.ResolveEdges(); err != nil {
		t.Fatalf("failed to resolve edges: %v", err)
	}

	// 4. Test FindSymbols (with FTS5 and LIKE)
	results, err := db.FindSymbols("ProcessPayment", 10)
	if err != nil {
		t.Fatalf("error finding symbols: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(results))
	}
	if results[0].Name != "ProcessPayment" {
		t.Errorf("expected ProcessPayment, got %s", results[0].Name)
	}

	// Test FTS5 docstring search
	docResults, err := db.FindSymbols("authorization", 10)
	if err != nil {
		t.Fatalf("error finding symbols by docstring: %v", err)
	}
	if len(docResults) == 0 {
		t.Errorf("expected FTS5 to find symbol by docstring 'authorization'")
	}

	// 5. Test GetSymbolByQualifiedName
	sym, symID, err := db.GetSymbolByQualifiedName("payment.ProcessPayment")
	if err != nil || sym == nil {
		t.Fatalf("failed to get symbol by qualified name: %v", err)
	}

	// 6. Test Neighborhood traversal
	subgraph, err := db.Neighborhood(symID, 2)
	if err != nil {
		t.Fatalf("failed to traverse neighborhood: %v", err)
	}

	if len(subgraph.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes in subgraph (caller + callee), got %d", len(subgraph.Nodes))
	}

	var foundAuthorize bool
	for _, node := range subgraph.Nodes {
		if node.Symbol.Name == "Authorize" {
			foundAuthorize = true
			if node.Distance != 1 {
				t.Errorf("expected distance 1 for Authorize, got %d", node.Distance)
			}
		}
	}
	if !foundAuthorize {
		t.Errorf("Authorize symbol was not found in neighborhood traversal")
	}

	// 7. Test Stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.TotalFiles != 2 {
		t.Errorf("expected 2 files, got %d", stats.TotalFiles)
	}
	if stats.TotalSymbols != 2 {
		t.Errorf("expected 2 symbols, got %d", stats.TotalSymbols)
	}
	if stats.TotalEdges != 1 {
		t.Errorf("expected 1 edge, got %d", stats.TotalEdges)
	}
}
