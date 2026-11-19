package graph

import (
	"fmt"
	"strings"
)

// ExportMermaid renders a neighborhood subgraph as a GitHub-compatible Mermaid diagram.
func ExportMermaid(subgraph *Subgraph) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph TD\n")
	sb.WriteString("  classDef target fill:#4f46e5,stroke:#818cf8,stroke-width:2px,color:#fff;\n")
	sb.WriteString("  classDef caller fill:#d97706,stroke:#fbbf24,stroke-width:1px,color:#fff;\n")
	sb.WriteString("  classDef callee fill:#0891b2,stroke:#22d3ee,stroke-width:1px,color:#fff;\n")
	sb.WriteString("  classDef dep fill:#334155,stroke:#64748b,stroke-width:1px,color:#fff;\n\n")

	nodeMap := make(map[string]string)
	for i, n := range subgraph.Nodes {
		nodeID := fmt.Sprintf("n%d", i)
		nodeMap[n.Symbol.Name] = nodeID
		nodeMap[n.Symbol.QualifiedName] = nodeID

		label := fmt.Sprintf("%s\\n[%s in %s]", n.Symbol.Name, n.Symbol.Kind, n.Symbol.FilePath)
		class := "dep"
		switch n.Role {
		case "target":
			class = "target"
		case "caller":
			class = "caller"
		case "callee":
			class = "callee"
		}

		sb.WriteString(fmt.Sprintf("  %s[\"%s\"]:::%s\n", nodeID, label, class))
	}

	sb.WriteString("\n")
	for _, e := range subgraph.Edges {
		srcID, hasSrc := nodeMap[e.CallerName]
		tgtID, hasTgt := nodeMap[e.CalleeName]
		if hasSrc && hasTgt && srcID != tgtID {
			sb.WriteString(fmt.Sprintf("  %s -->|%s| %s\n", srcID, e.EdgeType, tgtID))
		}
	}

	sb.WriteString("```")
	return sb.String()
}

// ExportDOT renders a subgraph in Graphviz DOT format.
func ExportDOT(subgraph *Subgraph) string {
	var sb strings.Builder
	sb.WriteString("digraph CallGraph {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\", fontsize=10];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\", fontsize=8];\n\n")

	nodeMap := make(map[string]string)
	for i, n := range subgraph.Nodes {
		nodeID := fmt.Sprintf("node_%d", i)
		nodeMap[n.Symbol.Name] = nodeID
		nodeMap[n.Symbol.QualifiedName] = nodeID

		fillColor := "#334155"
		fontColor := "#ffffff"
		switch n.Role {
		case "target":
			fillColor = "#4f46e5"
		case "caller":
			fillColor = "#d97706"
		case "callee":
			fillColor = "#0891b2"
		}

		label := fmt.Sprintf("%s\\n(%s)", n.Symbol.Name, n.Symbol.FilePath)
		sb.WriteString(fmt.Sprintf("  %s [label=\"%s\", fillcolor=\"%s\", fontcolor=\"%s\"];\n",
			nodeID, label, fillColor, fontColor))
	}

	sb.WriteString("\n")
	for _, e := range subgraph.Edges {
		srcID, hasSrc := nodeMap[e.CallerName]
		tgtID, hasTgt := nodeMap[e.CalleeName]
		if hasSrc && hasTgt && srcID != tgtID {
			sb.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\"];\n", srcID, tgtID, e.EdgeType))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}
