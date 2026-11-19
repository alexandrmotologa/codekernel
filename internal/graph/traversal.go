package graph

import (
	"fmt"

	"github.com/alexandrmotologa/codekernel/internal/parser"
)

// TraversalNode represents a symbol discovered in the neighborhood graph with distance metadata.
type TraversalNode struct {
	Symbol   parser.Symbol `json:"symbol"`
	SymbolID int64         `json:"symbol_id"`
	Distance int           `json:"distance"`
	Weight   float64       `json:"weight"`
	Role     string        `json:"role"` // "target", "caller", "callee", "dependency"
}

// Subgraph represents an extracted context neighborhood.
type Subgraph struct {
	EntryNode   TraversalNode   `json:"entry_node"`
	Nodes       []TraversalNode `json:"nodes"`
	Edges       []EdgeRecord    `json:"edges"`
	TotalTokens int             `json:"total_tokens"`
}

// Neighborhood traverses the call graph outward from entrySymbolID up to maxHops.
func (d *DB) Neighborhood(entrySymbolID int64, maxHops int) (*Subgraph, error) {
	if maxHops <= 0 {
		maxHops = 2
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Load entry symbol
	var entrySym parser.Symbol
	var kindStr string
	query := `
		SELECT s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
		       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
		       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.id = ?`
	err := d.sqldb.QueryRow(query, entrySymbolID).Scan(
		&entrySym.Name, &entrySym.QualifiedName, &kindStr, &entrySym.Signature, &entrySym.Docstring,
		&entrySym.FilePath, &entrySym.StartLine, &entrySym.EndLine, &entrySym.StartByte, &entrySym.EndByte,
		&entrySym.Receiver, &entrySym.Exported, &entrySym.Language, &entrySym.Body, &entrySym.TokenCount)
	if err != nil {
		return nil, fmt.Errorf("entry symbol not found: %w", err)
	}
	entrySym.Kind = parser.SymbolKind(kindStr)

	entryNode := TraversalNode{
		Symbol:   entrySym,
		SymbolID: entrySymbolID,
		Distance: 0,
		Weight:   1.0,
		Role:     "target",
	}

	visited := make(map[int64]bool)
	visited[entrySymbolID] = true

	nodes := []TraversalNode{entryNode}
	var edgeRecords []EdgeRecord
	currentLevel := []int64{entrySymbolID}

	for hop := 1; hop <= maxHops && len(currentLevel) > 0; hop++ {
		var nextLevel []int64
		weight := 1.0 / float64(hop+1)

		for _, currentID := range currentLevel {
			// 1. Find callers (inbound)
			callerRows, err := d.sqldb.Query(`
				SELECT DISTINCT s.id, s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
				       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
				       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
				FROM edges e
				JOIN symbols s ON e.caller_symbol_id = s.id
				JOIN files f ON s.file_id = f.id
				WHERE e.callee_symbol_id = ?`, currentID)
			if err == nil {
				for callerRows.Next() {
					var s parser.Symbol
					var sid int64
					var k string
					_ = callerRows.Scan(
						&sid, &s.Name, &s.QualifiedName, &k, &s.Signature, &s.Docstring,
						&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
						&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
					s.Kind = parser.SymbolKind(k)

					edgeRecords = append(edgeRecords, EdgeRecord{
						CallerSymbolID: &sid,
						CalleeSymbolID: &currentID,
						CallerName:     s.Name,
						CalleeName:     entrySym.Name,
						EdgeType:       "calls",
					})

					if !visited[sid] {
						visited[sid] = true
						nodes = append(nodes, TraversalNode{
							Symbol:   s,
							SymbolID: sid,
							Distance: hop,
							Weight:   weight,
							Role:     "caller",
						})
						nextLevel = append(nextLevel, sid)
					}
				}
				callerRows.Close()
			}

			// 2. Find callees (outbound)
			calleeRows, err := d.sqldb.Query(`
				SELECT DISTINCT s.id, s.name, s.qualified_name, s.kind, s.signature, COALESCE(s.docstring, ''),
				       f.path, s.start_line, s.end_line, s.start_byte, s.end_byte,
				       COALESCE(s.receiver, ''), s.exported, s.language, s.body, s.token_count
				FROM edges e
				JOIN symbols s ON e.callee_symbol_id = s.id
				JOIN files f ON s.file_id = f.id
				WHERE e.caller_symbol_id = ?`, currentID)
			if err == nil {
				for calleeRows.Next() {
					var s parser.Symbol
					var sid int64
					var k string
					_ = calleeRows.Scan(
						&sid, &s.Name, &s.QualifiedName, &k, &s.Signature, &s.Docstring,
						&s.FilePath, &s.StartLine, &s.EndLine, &s.StartByte, &s.EndByte,
						&s.Receiver, &s.Exported, &s.Language, &s.Body, &s.TokenCount)
					s.Kind = parser.SymbolKind(k)

					edgeRecords = append(edgeRecords, EdgeRecord{
						CallerSymbolID: &currentID,
						CalleeSymbolID: &sid,
						CallerName:     entrySym.Name,
						CalleeName:     s.Name,
						EdgeType:       "calls",
					})

					if !visited[sid] {
						visited[sid] = true
						nodes = append(nodes, TraversalNode{
							Symbol:   s,
							SymbolID: sid,
							Distance: hop,
							Weight:   weight,
							Role:     "callee",
						})
						nextLevel = append(nextLevel, sid)
					}
				}
				calleeRows.Close()
			}
		}

		currentLevel = nextLevel
	}

	totalTokens := 0
	for _, n := range nodes {
		totalTokens += n.Symbol.TokenCount
	}

	return &Subgraph{
		EntryNode:   entryNode,
		Nodes:       nodes,
		Edges:       edgeRecords,
		TotalTokens: totalTokens,
	}, nil
}
