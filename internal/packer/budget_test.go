package packer

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/parser"
)

func TestContextPackerBudgetStrictness(t *testing.T) {
	subgraph := &graph.Subgraph{
		EntryNode: graph.TraversalNode{
			Symbol: parser.Symbol{
				Name:          "ProcessOrder",
				QualifiedName: "order.ProcessOrder",
				Kind:          parser.KindFunction,
				Signature:     "func ProcessOrder(id string) error",
				Docstring:     "ProcessOrder coordinates checkout and validation",
				FilePath:      "pkg/order/process.go",
				StartLine:     10,
				EndLine:       30,
				Body:          "func ProcessOrder(id string) error {\n    // 50 tokens of logic...\n    return nil\n}",
				TokenCount:    50,
			},
			SymbolID: 1,
			Distance: 0,
			Weight:   1.0,
			Role:     "target",
		},
		Nodes: []graph.TraversalNode{
			{
				Symbol: parser.Symbol{
					Name:          "ProcessOrder",
					QualifiedName: "order.ProcessOrder",
					Kind:          parser.KindFunction,
					Signature:     "func ProcessOrder(id string) error",
					FilePath:      "pkg/order/process.go",
					StartLine:     10,
					EndLine:       30,
					Body:          "func ProcessOrder(id string) error {\n    return nil\n}",
					TokenCount:    50,
				},
				SymbolID: 1,
				Distance: 0,
				Weight:   1.0,
				Role:     "target",
			},
			{
				Symbol: parser.Symbol{
					Name:          "ValidateCart",
					QualifiedName: "cart.ValidateCart",
					Kind:          parser.KindFunction,
					Signature:     "func ValidateCart(items []Item) bool",
					FilePath:      "pkg/cart/validator.go",
					StartLine:     5,
					EndLine:       50,
					Body:          strings.Repeat("// long body\n", 40),
					TokenCount:    200,
				},
				SymbolID: 2,
				Distance: 1,
				Weight:   0.5,
				Role:     "callee",
			},
			{
				Symbol: parser.Symbol{
					Name:          "NotifyCustomer",
					QualifiedName: "notifier.NotifyCustomer",
					Kind:          parser.KindFunction,
					Signature:     "func NotifyCustomer(id string)",
					FilePath:      "pkg/notifier/email.go",
					StartLine:     1,
					EndLine:       80,
					Body:          strings.Repeat("// long notification body\n", 60),
					TokenCount:    300,
				},
				SymbolID: 3,
				Distance: 2,
				Weight:   0.33,
				Role:     "callee",
			},
		},
	}

	packer := NewContextPacker(4096, true)

	// Test 1: Tight budget (100 tokens). Target (50 tokens) + ValidateCart compressed (20 tokens) should fit.
	// NotifyCustomer should be omitted or compressed. Total tokens must be <= 100!
	packed := packer.Pack(subgraph, 100)
	if packed.UsedTokens > 100 {
		t.Fatalf("Used tokens (%d) strictly exceeded budget (100)", packed.UsedTokens)
	}
	if packed.TargetChunk.SymbolName != "order.ProcessOrder" {
		t.Errorf("expected target chunk ProcessOrder, got %s", packed.TargetChunk.SymbolName)
	}

	// Test 2: Formatting output
	formatter := NewFormatter()
	md := formatter.FormatMarkdown(packed)
	if !strings.Contains(md, "Code Context Subgraph: order.ProcessOrder") {
		t.Errorf("markdown missing header: %s", md)
	}

	jsonBytes, err := formatter.FormatJSON(packed)
	if err != nil {
		t.Fatalf("json format failed: %v", err)
	}
	if len(jsonBytes) == 0 {
		t.Errorf("empty json output")
	}
}
