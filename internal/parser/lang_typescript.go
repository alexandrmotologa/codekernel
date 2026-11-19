package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// TypeScriptExtractor extracts symbols and call sites from TypeScript and JavaScript ASTs.
type TypeScriptExtractor struct {
	isTSX bool
}

func (ts *TypeScriptExtractor) Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error) {
	var symbols []Symbol
	var calls []CallReference
	var imports []ImportReference

	// Recursive AST walk
	var walk func(node *sitter.Node, currentClass string)
	walk = func(node *sitter.Node, currentClass string) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "import_statement":
			imports = append(imports, ts.extractImport(node, content)...)

		case "export_statement":
			// Process inner exported declaration
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				ts.extractDeclaration(child, content, filePath, currentClass, true, &symbols)
			}
			return

		case "function_declaration", "generator_function_declaration":
			ts.extractDeclaration(node, content, filePath, currentClass, false, &symbols)

		case "lexical_declaration", "variable_declaration":
			ts.extractVariableFunctions(node, content, filePath, false, &symbols)

		case "class_declaration", "abstract_class_declaration":
			nameNode := node.ChildByFieldName("name")
			className := ""
			if nameNode != nil {
				className = nameNode.Content(content)
			}
			sym := ts.extractClass(node, content, filePath, className)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			// Process methods inside class body
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
					methodNode := bodyNode.NamedChild(i)
					if methodNode.Type() == "method_definition" {
						methodSym := ts.extractMethod(methodNode, content, filePath, className)
						if methodSym != nil {
							symbols = append(symbols, *methodSym)
						}
					}
				}
			}
			return

		case "interface_declaration":
			sym := ts.extractInterface(node, content, filePath)
			if sym != nil {
				symbols = append(symbols, *sym)
			}

		case "type_alias_declaration":
			sym := ts.extractTypeAlias(node, content, filePath)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i), currentClass)
		}
	}

	walk(root, "")
	calls = ts.extractCalls(root, content, filePath)

	return symbols, calls, imports, nil
}

func (ts *TypeScriptExtractor) extractDeclaration(node *sitter.Node, content []byte, filePath, currentClass string, exported bool, symbols *[]Symbol) {
	switch node.Type() {
	case "function_declaration", "generator_function_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return
		}
		name := nameNode.Content(content)
		doc := ts.extractDocstring(node, content)

		paramsNode := node.ChildByFieldName("parameters")
		sig := "function " + name
		if paramsNode != nil {
			sig += paramsNode.Content(content)
		} else {
			sig += "()"
		}
		if retNode := node.ChildByFieldName("return_type"); retNode != nil {
			sig += ": " + retNode.Content(content)
		}

		body := node.Content(content)
		*symbols = append(*symbols, Symbol{
			Name:          name,
			QualifiedName: name,
			Kind:          KindFunction,
			Signature:     sig,
			Docstring:     doc,
			FilePath:      filePath,
			StartLine:     int(node.StartPoint().Row) + 1,
			EndLine:       int(node.EndPoint().Row) + 1,
			StartByte:     node.StartByte(),
			EndByte:       node.EndByte(),
			Exported:      exported,
			Language:      "typescript",
			Body:          body,
			TokenCount:    EstimateTokens(body),
		})

	case "class_declaration", "abstract_class_declaration":
		nameNode := node.ChildByFieldName("name")
		className := ""
		if nameNode != nil {
			className = nameNode.Content(content)
		}
		sym := ts.extractClass(node, content, filePath, className)
		if sym != nil {
			sym.Exported = exported
			*symbols = append(*symbols, *sym)
		}
		bodyNode := node.ChildByFieldName("body")
		if bodyNode != nil {
			for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
				m := bodyNode.NamedChild(i)
				if m.Type() == "method_definition" {
					methodSym := ts.extractMethod(m, content, filePath, className)
					if methodSym != nil {
						methodSym.Exported = exported
						*symbols = append(*symbols, *methodSym)
					}
				}
			}
		}

	case "interface_declaration":
		sym := ts.extractInterface(node, content, filePath)
		if sym != nil {
			sym.Exported = exported
			*symbols = append(*symbols, *sym)
		}

	case "lexical_declaration", "variable_declaration":
		ts.extractVariableFunctions(node, content, filePath, exported, symbols)

	case "type_alias_declaration":
		sym := ts.extractTypeAlias(node, content, filePath)
		if sym != nil {
			sym.Exported = exported
			*symbols = append(*symbols, *sym)
		}
	}
}

func (ts *TypeScriptExtractor) extractVariableFunctions(node *sitter.Node, content []byte, filePath string, exported bool, symbols *[]Symbol) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		decl := node.NamedChild(i)
		if decl.Type() == "variable_declarator" {
			nameNode := decl.ChildByFieldName("name")
			valNode := decl.ChildByFieldName("value")
			if nameNode != nil && valNode != nil {
				if valNode.Type() == "arrow_function" || valNode.Type() == "function_expression" {
					name := nameNode.Content(content)
					doc := ts.extractDocstring(node, content)
					sig := "const " + name + " = "
					if paramsNode := valNode.ChildByFieldName("parameters"); paramsNode != nil {
						sig += paramsNode.Content(content) + " => ..."
					} else {
						sig += "() => ..."
					}

					body := decl.Content(content)
					*symbols = append(*symbols, Symbol{
						Name:          name,
						QualifiedName: name,
						Kind:          KindFunction,
						Signature:     sig,
						Docstring:     doc,
						FilePath:      filePath,
						StartLine:     int(node.StartPoint().Row) + 1,
						EndLine:       int(node.EndPoint().Row) + 1,
						StartByte:     node.StartByte(),
						EndByte:       node.EndByte(),
						Exported:      exported,
						Language:      "typescript",
						Body:          body,
						TokenCount:    EstimateTokens(body),
					})
				}
			}
		}
	}
}

func (ts *TypeScriptExtractor) extractClass(node *sitter.Node, content []byte, filePath, className string) *Symbol {
	if className == "" {
		return nil
	}
	doc := ts.extractDocstring(node, content)
	sig := "class " + className

	heritageNode := node.ChildByFieldName("heritage")
	if heritageNode != nil {
		sig += " " + heritageNode.Content(content)
	}

	body := node.Content(content)
	return &Symbol{
		Name:          className,
		QualifiedName: className,
		Kind:          KindClass,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      false,
		Language:      "typescript",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (ts *TypeScriptExtractor) extractMethod(node *sitter.Node, content []byte, filePath, className string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := ts.extractDocstring(node, content)

	sig := name
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		sig += paramsNode.Content(content)
	} else {
		sig += "()"
	}
	if retNode := node.ChildByFieldName("return_type"); retNode != nil {
		sig += ": " + retNode.Content(content)
	}

	body := node.Content(content)
	qualified := className + "." + name

	return &Symbol{
		Name:          name,
		QualifiedName: qualified,
		Kind:          KindMethod,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Receiver:      className,
		Exported:      false,
		Language:      "typescript",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (ts *TypeScriptExtractor) extractInterface(node *sitter.Node, content []byte, filePath string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := ts.extractDocstring(node, content)
	sig := "interface " + name

	body := node.Content(content)
	return &Symbol{
		Name:          name,
		QualifiedName: name,
		Kind:          KindInterface,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      false,
		Language:      "typescript",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (ts *TypeScriptExtractor) extractTypeAlias(node *sitter.Node, content []byte, filePath string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := ts.extractDocstring(node, content)
	sig := "type " + name

	body := node.Content(content)
	return &Symbol{
		Name:          name,
		QualifiedName: name,
		Kind:          KindTypeAlias,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     int(node.StartPoint().Row) + 1,
		EndLine:       int(node.EndPoint().Row) + 1,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      false,
		Language:      "typescript",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (ts *TypeScriptExtractor) extractImport(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	sourceNode := node.ChildByFieldName("source")
	if sourceNode == nil {
		return results
	}
	path := strings.Trim(sourceNode.Content(content), `"'`)

	var importedSymbols []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "import_clause" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				sub := child.NamedChild(j)
				if sub.Type() == "named_imports" {
					for k := 0; k < int(sub.NamedChildCount()); k++ {
						spec := sub.NamedChild(k)
						if spec.Type() == "import_specifier" {
							nameNode := spec.ChildByFieldName("name")
							if nameNode != nil {
								importedSymbols = append(importedSymbols, nameNode.Content(content))
							}
						}
					}
				} else if sub.Type() == "identifier" {
					importedSymbols = append(importedSymbols, sub.Content(content))
				}
			}
		}
	}

	results = append(results, ImportReference{
		Path:    path,
		Symbols: importedSymbols,
	})
	return results
}

func (ts *TypeScriptExtractor) extractCalls(root *sitter.Node, content []byte, filePath string) []CallReference {
	var calls []CallReference

	var traverse func(node *sitter.Node, currentCaller string)
	traverse = func(n *sitter.Node, currentCaller string) {
		if n == nil {
			return
		}

		newCaller := currentCaller
		if n.Type() == "function_declaration" || n.Type() == "method_definition" {
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
				} else if fnNode.Type() == "member_expression" {
					propNode := fnNode.ChildByFieldName("property")
					objNode := fnNode.ChildByFieldName("object")
					if propNode != nil {
						calleeName = propNode.Content(content)
					}
					if objNode != nil {
						receiver = objNode.Content(content)
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

func (ts *TypeScriptExtractor) extractDocstring(node *sitter.Node, content []byte) string {
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
