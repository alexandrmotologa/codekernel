package parser

import (
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
)

// GoExtractor extracts declarations and call sites from Go AST.
type GoExtractor struct{}

func (g *GoExtractor) Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error) {
	var symbols []Symbol
	var calls []CallReference
	var imports []ImportReference

	packageName := ""

	// First pass: locate package name
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "package_clause" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				sub := child.NamedChild(j)
				if sub.Type() == "package_identifier" {
					packageName = sub.Content(content)
					break
				}
			}
		}
	}

	// Traverse declarations
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		switch child.Type() {
		case "import_declaration":
			imports = append(imports, g.extractImports(child, content)...)
		case "function_declaration":
			sym := g.extractFunction(child, content, filePath, packageName)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
		case "method_declaration":
			sym := g.extractMethod(child, content, filePath, packageName)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
		case "type_declaration":
			typeSymbols := g.extractType(child, content, filePath, packageName)
			symbols = append(symbols, typeSymbols...)
		}
	}

	// Extract calls across the whole file, attributing to enclosing functions
	calls = g.extractCalls(root, content, filePath)

	return symbols, calls, imports, nil
}

func (g *GoExtractor) extractImports(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "import_spec" {
			pathNode := child.ChildByFieldName("path")
			aliasNode := child.ChildByFieldName("name")

			path := ""
			if pathNode != nil {
				path = strings.Trim(pathNode.Content(content), `"`)
			}
			alias := ""
			if aliasNode != nil {
				alias = aliasNode.Content(content)
			}
			if path != "" {
				results = append(results, ImportReference{
					Path:  path,
					Alias: alias,
				})
			}
		} else if child.Type() == "import_spec_list" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() == "import_spec" {
					pathNode := spec.ChildByFieldName("path")
					aliasNode := spec.ChildByFieldName("name")

					path := ""
					if pathNode != nil {
						path = strings.Trim(pathNode.Content(content), `"`)
					}
					alias := ""
					if aliasNode != nil {
						alias = aliasNode.Content(content)
					}
					if path != "" {
						results = append(results, ImportReference{
							Path:  path,
							Alias: alias,
						})
					}
				}
			}
		}
	}
	return results
}

func (g *GoExtractor) extractFunction(node *sitter.Node, content []byte, filePath, pkg string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := g.extractDocstring(node, content)

	sigNode := node.ChildByFieldName("parameters")
	sig := ""
	if sigNode != nil {
		sig = "func " + name + sigNode.Content(content)
		if resNode := node.ChildByFieldName("result"); resNode != nil {
			sig += " " + resNode.Content(content)
		}
	} else {
		sig = "func " + name + "()"
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	qualified := name
	if pkg != "" {
		qualified = pkg + "." + name
	}

	exported := false
	if len(name) > 0 {
		exported = unicode.IsUpper(rune(name[0]))
	}

	return &Symbol{
		Name:          name,
		QualifiedName: qualified,
		Kind:          KindFunction,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     startRow,
		EndLine:       endRow,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      exported,
		Language:      "go",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (g *GoExtractor) extractMethod(node *sitter.Node, content []byte, filePath, pkg string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	recvNode := node.ChildByFieldName("receiver")
	if nameNode == nil || recvNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	recvText := recvNode.Content(content)
	cleanRecv := strings.Trim(recvText, "()")
	receiverType := cleanRecv
	parts := strings.Fields(cleanRecv)
	if len(parts) >= 2 {
		receiverType = parts[len(parts)-1]
	}
	receiverType = strings.TrimPrefix(receiverType, "*")

	doc := g.extractDocstring(node, content)

	sigNode := node.ChildByFieldName("parameters")
	sig := "func (" + cleanRecv + ") " + name
	if sigNode != nil {
		sig += sigNode.Content(content)
		if resNode := node.ChildByFieldName("result"); resNode != nil {
			sig += " " + resNode.Content(content)
		}
	} else {
		sig += "()"
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	qualified := receiverType + "." + name
	if pkg != "" {
		qualified = pkg + "." + qualified
	}

	exported := false
	if len(name) > 0 {
		exported = unicode.IsUpper(rune(name[0]))
	}

	return &Symbol{
		Name:          name,
		QualifiedName: qualified,
		Kind:          KindMethod,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     startRow,
		EndLine:       endRow,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Receiver:      receiverType,
		Exported:      exported,
		Language:      "go",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (g *GoExtractor) extractType(node *sitter.Node, content []byte, filePath, pkg string) []Symbol {
	var symbols []Symbol
	doc := g.extractDocstring(node, content)

	for i := 0; i < int(node.NamedChildCount()); i++ {
		spec := node.NamedChild(i)
		if spec.Type() != "type_spec" {
			continue
		}

		nameNode := spec.ChildByFieldName("name")
		typeNode := spec.ChildByFieldName("type")
		if nameNode == nil {
			continue
		}

		name := nameNode.Content(content)
		kind := KindTypeAlias
		if typeNode != nil {
			switch typeNode.Type() {
			case "struct_type":
				kind = KindStruct
			case "interface_type":
				kind = KindInterface
			}
		}

		sig := "type " + name
		if typeNode != nil {
			switch kind {
			case KindStruct:
				sig += " struct"
			case KindInterface:
				sig += " interface"
			default:
				sig += " " + typeNode.Content(content)
			}
		}

		body := spec.Content(content)
		startRow := int(node.StartPoint().Row) + 1
		endRow := int(node.EndPoint().Row) + 1

		qualified := name
		if pkg != "" {
			qualified = pkg + "." + name
		}

		exported := false
		if len(name) > 0 {
			exported = unicode.IsUpper(rune(name[0]))
		}

		symbols = append(symbols, Symbol{
			Name:          name,
			QualifiedName: qualified,
			Kind:          kind,
			Signature:     sig,
			Docstring:     doc,
			FilePath:      filePath,
			StartLine:     startRow,
			EndLine:       endRow,
			StartByte:     node.StartByte(),
			EndByte:       node.EndByte(),
			Exported:      exported,
			Language:      "go",
			Body:          body,
			TokenCount:    EstimateTokens(body),
		})
	}
	return symbols
}

func (g *GoExtractor) extractCalls(root *sitter.Node, content []byte, filePath string) []CallReference {
	var calls []CallReference

	// Helper to find caller scope
	var traverse func(node *sitter.Node, currentCaller string)
	traverse = func(n *sitter.Node, currentCaller string) {
		if n == nil {
			return
		}

		// Update caller scope if entering a function or method
		newCaller := currentCaller
		if n.Type() == "function_declaration" || n.Type() == "method_declaration" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				newCaller = nameNode.Content(content)
				if n.Type() == "method_declaration" {
					if recvNode := n.ChildByFieldName("receiver"); recvNode != nil {
						recvText := strings.Trim(recvNode.Content(content), "()* ")
						parts := strings.Fields(recvText)
						if len(parts) > 0 {
							recvType := strings.TrimPrefix(parts[len(parts)-1], "*")
							newCaller = recvType + "." + newCaller
						}
					}
				}
			}
		}

		if n.Type() == "call_expression" {
			fnNode := n.ChildByFieldName("function")
			if fnNode != nil {
				calleeName := ""
				receiver := ""
				isMethod := false

				if fnNode.Type() == "identifier" {
					calleeName = fnNode.Content(content)
				} else if fnNode.Type() == "selector_expression" {
					fieldNode := fnNode.ChildByFieldName("field")
					operandNode := fnNode.ChildByFieldName("operand")
					if fieldNode != nil {
						calleeName = fieldNode.Content(content)
					}
					if operandNode != nil {
						receiver = operandNode.Content(content)
						isMethod = true
					}
				}

				if calleeName != "" {
					calls = append(calls, CallReference{
						CallerName:     newCaller,
						CalleeName:     calleeName,
						CalleeReceiver: receiver,
						FilePath:       filePath,
						Line:           int(n.StartPoint().Row) + 1,
						ByteOffset:     n.StartByte(),
						IsMethodCall:   isMethod,
					})
				}
			}
		}

		for i := 0; i < int(n.NamedChildCount()); i++ {
			traverse(n.NamedChild(i), newCaller)
		}
	}

	traverse(root, "")
	return calls
}

func (g *GoExtractor) extractDocstring(node *sitter.Node, content []byte) string {
	// Look at preceding siblings for comments
	prev := node.PrevSibling()
	var comments []string
	for prev != nil {
		if prev.Type() == "comment" {
			comments = append([]string{prev.Content(content)}, comments...)
		} else if strings.TrimSpace(prev.Content(content)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return strings.Join(comments, "\n")
}
