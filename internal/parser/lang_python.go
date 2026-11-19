package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// PythonExtractor extracts declarations and call sites from Python AST.
type PythonExtractor struct{}

func (py *PythonExtractor) Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error) {
	var symbols []Symbol
	var calls []CallReference
	var imports []ImportReference

	var walk func(node *sitter.Node, currentClass string)
	walk = func(node *sitter.Node, currentClass string) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "import_statement":
			imports = append(imports, py.extractImport(node, content)...)

		case "import_from_statement":
			imports = append(imports, py.extractImportFrom(node, content)...)

		case "function_definition":
			sym := py.extractFunction(node, content, filePath, currentClass)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			// Recurse into function body for nested definitions
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
					walk(bodyNode.NamedChild(i), currentClass)
				}
			}
			return

		case "decorated_definition":
			// Contains decorator and target definition
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child.Type() == "function_definition" {
					sym := py.extractFunction(child, content, filePath, currentClass)
					if sym != nil {
						symbols = append(symbols, *sym)
					}
					return
				} else if child.Type() == "class_definition" {
					walk(child, currentClass)
					return
				}
			}

		case "class_definition":
			nameNode := node.ChildByFieldName("name")
			className := ""
			if nameNode != nil {
				className = nameNode.Content(content)
			}
			sym := py.extractClass(node, content, filePath, className)
			if sym != nil {
				symbols = append(symbols, *sym)
			}
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
					walk(bodyNode.NamedChild(i), className)
				}
			}
			return
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i), currentClass)
		}
	}

	walk(root, "")
	calls = py.extractCalls(root, content, filePath)

	return symbols, calls, imports, nil
}

func (py *PythonExtractor) extractFunction(node *sitter.Node, content []byte, filePath, currentClass string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := py.extractDocstringFromBody(node, content)

	paramsNode := node.ChildByFieldName("parameters")
	sig := "def " + name
	if paramsNode != nil {
		sig += paramsNode.Content(content)
	} else {
		sig += "()"
	}
	if retNode := node.ChildByFieldName("return_type"); retNode != nil {
		sig += " -> " + retNode.Content(content)
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	kind := KindFunction
	qualified := name
	receiver := ""
	if currentClass != "" {
		kind = KindMethod
		qualified = currentClass + "." + name
		receiver = currentClass
	}

	exported := !strings.HasPrefix(name, "_")

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
		Language:      "python",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (py *PythonExtractor) extractClass(node *sitter.Node, content []byte, filePath, className string) *Symbol {
	if className == "" {
		return nil
	}
	doc := py.extractDocstringFromBody(node, content)
	sig := "class " + className

	superNode := node.ChildByFieldName("superclasses")
	if superNode != nil {
		sig += superNode.Content(content)
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1
	exported := !strings.HasPrefix(className, "_")

	return &Symbol{
		Name:          className,
		QualifiedName: className,
		Kind:          KindClass,
		Signature:     sig,
		Docstring:     doc,
		FilePath:      filePath,
		StartLine:     startRow,
		EndLine:       endRow,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Exported:      exported,
		Language:      "python",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (py *PythonExtractor) extractDocstringFromBody(node *sitter.Node, content []byte) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil || bodyNode.NamedChildCount() == 0 {
		return ""
	}
	firstStmt := bodyNode.NamedChild(0)
	if firstStmt.Type() == "expression_statement" && firstStmt.NamedChildCount() > 0 {
		expr := firstStmt.NamedChild(0)
		if expr.Type() == "string" {
			raw := expr.Content(content)
			raw = strings.Trim(raw, `"'`)
			return strings.TrimSpace(raw)
		}
	}
	return ""
}

func (py *PythonExtractor) extractImport(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "dotted_name" {
			results = append(results, ImportReference{
				Path: child.Content(content),
			})
		}
	}
	return results
}

func (py *PythonExtractor) extractImportFrom(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	moduleNode := node.ChildByFieldName("module_name")
	module := ""
	if moduleNode != nil {
		module = moduleNode.Content(content)
	}

	var symbols []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "dotted_name" && child != moduleNode {
			symbols = append(symbols, child.Content(content))
		}
	}

	results = append(results, ImportReference{
		Path:    module,
		Symbols: symbols,
	})
	return results
}

func (py *PythonExtractor) extractCalls(root *sitter.Node, content []byte, filePath string) []CallReference {
	var calls []CallReference

	var traverse func(node *sitter.Node, currentCaller string)
	traverse = func(n *sitter.Node, currentCaller string) {
		if n == nil {
			return
		}

		newCaller := currentCaller
		if n.Type() == "function_definition" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				newCaller = nameNode.Content(content)
			}
		}

		if n.Type() == "call" {
			fnNode := n.ChildByFieldName("function")
			if fnNode != nil {
				calleeName := ""
				receiver := ""
				isMethod := false

				if fnNode.Type() == "identifier" {
					calleeName = fnNode.Content(content)
				} else if fnNode.Type() == "attribute" {
					objNode := fnNode.ChildByFieldName("object")
					attrNode := fnNode.ChildByFieldName("attribute")
					if attrNode != nil {
						calleeName = attrNode.Content(content)
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
