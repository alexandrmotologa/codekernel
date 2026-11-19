package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// RustExtractor extracts declarations and call sites from Rust AST.
type RustExtractor struct{}

func (rs *RustExtractor) Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error) {
	var symbols []Symbol
	var calls []CallReference
	var imports []ImportReference

	var walk func(node *sitter.Node, currentScope string)
	walk = func(node *sitter.Node, currentScope string) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "use_declaration":
			imports = append(imports, rs.extractUse(node, content)...)

		case "function_item":
			sym := rs.extractFunction(node, content, filePath, currentScope)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			return

		case "struct_item":
			sym := rs.extractItem(node, content, filePath, KindStruct)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			return

		case "enum_item":
			sym := rs.extractItem(node, content, filePath, KindEnum)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			return

		case "trait_item":
			sym := rs.extractItem(node, content, filePath, KindTrait)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			return

		case "impl_item":
			typeNode := node.ChildByFieldName("type")
			targetType := ""
			if typeNode != nil {
				targetType = typeNode.Content(content)
			}

			traitNode := node.ChildByFieldName("trait")
			traitName := ""
			if traitNode != nil {
				traitName = traitNode.Content(content)
			}

			scope := targetType
			if traitName != "" {
				scope = targetType + " as " + traitName
			}

			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
					child := bodyNode.NamedChild(i)
					if child.Type() == "function_item" {
						sym := rs.extractFunction(child, content, filePath, scope)
						if sym != nil {
							sym.Receiver = targetType
							symbols = append(symbols, *sym)
						}
					}
				}
			}
			return
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i), currentScope)
		}
	}

	walk(root, "")
	calls = rs.extractCalls(root, content, filePath)

	return symbols, calls, imports, nil
}

func (rs *RustExtractor) extractFunction(node *sitter.Node, content []byte, filePath, currentScope string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := rs.extractDocstring(node, content)

	sig := "fn " + name
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		sig += paramsNode.Content(content)
	} else {
		sig += "()"
	}
	if retNode := node.ChildByFieldName("return_type"); retNode != nil {
		sig += " " + retNode.Content(content)
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	kind := KindFunction
	qualified := name
	receiver := ""
	if currentScope != "" {
		kind = KindMethod
		qualified = currentScope + "::" + name
		receiver = currentScope
	}

	exported := false
	visNode := node.ChildByFieldName("visibility")
	if visNode != nil && strings.Contains(visNode.Content(content), "pub") {
		exported = true
	}

	return &Symbol{
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
		Receiver:      receiver,
		Exported:      exported,
		Language:      "rust",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (rs *RustExtractor) extractItem(node *sitter.Node, content []byte, filePath string, kind SymbolKind) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := rs.extractDocstring(node, content)
	sig := string(kind) + " " + name

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	exported := false
	visNode := node.ChildByFieldName("visibility")
	if visNode != nil && strings.Contains(visNode.Content(content), "pub") {
		exported = true
	}

	return &Symbol{
		Name:          name,
		QualifiedName: name,
		Kind:          kind,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     startRow,
		EndLine:       endRow,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      exported,
		Language:      "rust",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (rs *RustExtractor) extractUse(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	argNode := node.ChildByFieldName("argument")
	if argNode != nil {
		path := argNode.Content(content)
		path = strings.TrimSuffix(path, ";")
		results = append(results, ImportReference{
			Path: path,
		})
	}
	return results
}

func (rs *RustExtractor) extractCalls(root *sitter.Node, content []byte, filePath string) []CallReference {
	var calls []CallReference

	var traverse func(node *sitter.Node, currentCaller string)
	traverse = func(n *sitter.Node, currentCaller string) {
		if n == nil {
			return
		}

		newCaller := currentCaller
		if n.Type() == "function_item" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				newCaller = nameNode.Content(content)
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
				} else if fnNode.Type() == "field_expression" {
					valNode := fnNode.ChildByFieldName("value")
					fieldNode := fnNode.ChildByFieldName("field")
					if fieldNode != nil {
						calleeName = fieldNode.Content(content)
					}
					if valNode != nil {
						receiver = valNode.Content(content)
						isMethod = true
					}
				} else if fnNode.Type() == "scoped_identifier" {
					nameNode := fnNode.ChildByFieldName("name")
					pathNode := fnNode.ChildByFieldName("path")
					if nameNode != nil {
						calleeName = nameNode.Content(content)
					}
					if pathNode != nil {
						receiver = pathNode.Content(content)
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

func (rs *RustExtractor) extractDocstring(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	var comments []string
	for prev != nil {
		if prev.Type() == "line_comment" || prev.Type() == "block_comment" {
			comments = append([]string{prev.Content(content)}, comments...)
		} else if strings.TrimSpace(prev.Content(content)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return strings.Join(comments, "\n")
}
