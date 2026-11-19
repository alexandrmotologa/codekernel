package packer

import (
	"sort"

	"github.com/alexandrmotologa/codekernel/internal/chunker"
	"github.com/alexandrmotologa/codekernel/internal/graph"
)

// PackedContext represents the surgical context subgraph prepared within exact token limits.
type PackedContext struct {
	EntrySymbol     string          `json:"entry_symbol"`
	Budget          int             `json:"budget"`
	UsedTokens      int             `json:"used_tokens"`
	TargetChunk     chunker.Chunk   `json:"target_chunk"`
	CallerChunks    []chunker.Chunk `json:"caller_chunks"`
	CalleeChunks    []chunker.Chunk `json:"callee_chunks"`
	OtherChunks     []chunker.Chunk `json:"other_chunks"`
	CompressedCount int             `json:"compressed_count"`
	DroppedCount    int             `json:"dropped_count"`
}

// ContextPacker executes knapsack packing over neighborhood subgraphs.
type ContextPacker struct {
	DefaultBudget   int
	CompressCallees bool
}

// NewContextPacker creates a new context packer.
func NewContextPacker(defaultBudget int, compressCallees bool) *ContextPacker {
	if defaultBudget <= 0 {
		defaultBudget = 4096
	}
	return &ContextPacker{
		DefaultBudget:   defaultBudget,
		CompressCallees: compressCallees,
	}
}

// Pack extracts the optimal context subgraph given a neighborhood and token limit.
func (cp *ContextPacker) Pack(subgraph *graph.Subgraph, budget int) *PackedContext {
	if budget <= 0 {
		budget = cp.DefaultBudget
	}

	result := &PackedContext{
		EntrySymbol: subgraph.EntryNode.Symbol.QualifiedName,
		Budget:      budget,
	}

	remainingBudget := budget

	// 1. Pack entry symbol (Hop 0)
	targetSym := subgraph.EntryNode.Symbol
	targetFullChunk := chunker.CompressSymbolBody(targetSym, false)

	if targetFullChunk.TokenCount <= remainingBudget {
		result.TargetChunk = targetFullChunk
		remainingBudget -= targetFullChunk.TokenCount
		result.UsedTokens += targetFullChunk.TokenCount
	} else {
		// Even target doesn't fit, use compressed signature
		comp := chunker.CompressSignature(targetSym)
		result.TargetChunk = comp
		remainingBudget -= comp.TokenCount
		result.UsedTokens += comp.TokenCount
		result.CompressedCount++
	}

	// 2. Separate remaining nodes by distance / role
	var candidates []graph.TraversalNode
	for _, n := range subgraph.Nodes {
		if n.Distance > 0 {
			candidates = append(candidates, n)
		}
	}

	// Sort candidates by weight descending (distance 1 first, then distance 2), then by exported status
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Weight != candidates[j].Weight {
			return candidates[i].Weight > candidates[j].Weight
		}
		if candidates[i].Symbol.Exported != candidates[j].Symbol.Exported {
			return candidates[i].Symbol.Exported
		}
		return candidates[i].Symbol.TokenCount < candidates[j].Symbol.TokenCount
	})

	// 3. Greedy knapsack packing
	for _, candidate := range candidates {
		sym := candidate.Symbol
		// If hop >= 2 or compressCallees is true, prefer compressed signature first
		preferCompressed := candidate.Distance >= 2 || (cp.CompressCallees && candidate.Role == "callee")

		if !preferCompressed {
			fullChunk := chunker.CompressSymbolBody(sym, false)
			if fullChunk.TokenCount <= remainingBudget {
				remainingBudget -= fullChunk.TokenCount
				result.UsedTokens += fullChunk.TokenCount
				cp.addChunkByRole(result, candidate.Role, fullChunk)
				continue
			}
		}

		// Try compressed signature
		compChunk := chunker.CompressSignature(sym)
		if compChunk.TokenCount <= remainingBudget {
			remainingBudget -= compChunk.TokenCount
			result.UsedTokens += compChunk.TokenCount
			result.CompressedCount++
			cp.addChunkByRole(result, candidate.Role, compChunk)
		} else {
			result.DroppedCount++
		}
	}

	return result
}

func (cp *ContextPacker) addChunkByRole(res *PackedContext, role string, c chunker.Chunk) {
	switch role {
	case "caller":
		res.CallerChunks = append(res.CallerChunks, c)
	case "callee":
		res.CalleeChunks = append(res.CalleeChunks, c)
	default:
		res.OtherChunks = append(res.OtherChunks, c)
	}
}

// AllChunks returns a flat slice of all packed chunks.
func (p *PackedContext) AllChunks() []chunker.Chunk {
	var all []chunker.Chunk
	if p.TargetChunk.Content != "" {
		all = append(all, p.TargetChunk)
	}
	all = append(all, p.CallerChunks...)
	all = append(all, p.CalleeChunks...)
	all = append(all, p.OtherChunks...)
	return all
}
