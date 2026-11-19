package chunker

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestChunkFileAndSignatures(t *testing.T) {
	code := `package mathutils

// Add calculates the sum of two integers
func Add(a int, b int) int {
	return a + b
}

// Multiply calculates product
func Multiply(a int, b int) int {
	// multiple lines of logic
	res := 0
	for i := 0; i < b; i++ {
		res += a
	}
	return res
}
`
	engine := parser.NewEngine()
	pf, err := engine.Parse("math.go", []byte(code))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	chunker := NewChunker()
	chunks := chunker.ChunkFile(pf)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify no function was cut in half
	for _, c := range chunks {
		if c.Kind == parser.KindFunction {
			if !strings.Contains(c.Content, "func ") {
				t.Errorf("chunk missing function header: %s", c.Content)
			}
			if !strings.Contains(c.Content, "}") {
				t.Errorf("chunk missing function closing brace: %s", c.Content)
			}
		}
	}

	// Test signature compression
	comp := CompressSignature(pf.Symbols[1])
	if !strings.Contains(comp.Content, "body omitted") {
		t.Errorf("expected body omitted stub, got: %s", comp.Content)
	}
	if comp.TokenCount >= pf.Symbols[1].TokenCount {
		t.Errorf("compressed tokens (%d) should be strictly less than full body tokens (%d)",
			comp.TokenCount, pf.Symbols[1].TokenCount)
	}
}
