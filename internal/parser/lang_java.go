package parser

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JavaExtractor extracts declarations and call sites from Java AST.
type JavaExtractor struct{}

func (j *JavaExtractor) Parse(root *sitter.Node, content []byte, filePath string) ([]Symbol, []CallReference, []ImportReference, error) {
	var symbols []Symbol
	var calls []CallReference
	var imports []ImportReference

	packageName := ""

	// Extract package declaration
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "package_declaration" {
			for k := 0; k < int(child.NamedChildCount()); k++ {
				sub := child.NamedChild(k)
				if sub.Type() == "scoped_identifier" || sub.Type() == "identifier" {
					packageName = sub.Content(content)
					break
				}
			}
		}
	}

	var walk func(node *sitter.Node, currentClass string)
	walk = func(node *sitter.Node, currentClass string) {
		if node == nil {
			return
		}

		switch node.Type() {
		case "import_declaration":
			imports = append(imports, j.extractImport(node, content)...)

		case "class_declaration", "interface_declaration", "record_declaration", "enum_declaration":
			nameNode := node.ChildByFieldName("name")
			className := ""
			if nameNode != nil {
				className = nameNode.Content(content)
			}
			sym := j.extractTypeDecl(node, content, filePath, packageName, className)
			if sym != nil {
				symbols = append(symbols, *sym)
			}

			// Traverse inner members
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
					child := bodyNode.NamedChild(i)
					if child.Type() == "method_declaration" || child.Type() == "constructor_declaration" {
						methodSym := j.extractMethod(child, content, filePath, packageName, className)
						if methodSym != nil {
							symbols = append(symbols, *methodSym)
						}
					} else {
						walk(child, className)
					}
				}
			}
			return
		}

		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i), currentClass)
		}
	}

	walk(root, "")
	calls = j.extractCalls(root, content, filePath)

	return symbols, calls, imports, nil
}

func (j *JavaExtractor) extractTypeDecl(node *sitter.Node, content []byte, filePath, pkg, name string) *Symbol {
	if name == "" {
		return nil
	}

	kind := KindClass
	switch node.Type() {
	case "interface_declaration":
		kind = KindInterface
	case "record_declaration":
		kind = KindClass
	case "enum_declaration":
		kind = KindEnum
	}

	doc := j.extractDocstring(node, content)
	sig := string(kind) + " " + name

	if superNode := node.ChildByFieldName("superclass"); superNode != nil {
		sig += " extends " + superNode.Content(content)
	}
	if ifacesNode := node.ChildByFieldName("interfaces"); ifacesNode != nil {
		sig += " implements " + ifacesNode.Content(content)
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	qualified := name
	if pkg != "" {
		qualified = pkg + "." + name
	}

	exported := false
	for i := 0; i < int(node.NamedChildCount()); i++ {
		ch := node.NamedChild(i)
		if ch.Type() == "modifiers" && strings.Contains(ch.Content(content), "public") {
			exported = true
			break
		}
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
		Exported:      exported,
		Language:      "java",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (j *JavaExtractor) extractMethod(node *sitter.Node, content []byte, filePath, pkg, className string) *Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(content)
	doc := j.extractDocstring(node, content)

	sig := name
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		sig += paramsNode.Content(content)
	} else {
		sig += "()"
	}
	if retNode := node.ChildByFieldName("type"); retNode != nil {
		sig = retNode.Content(content) + " " + sig
	}

	body := node.Content(content)
	startRow := int(node.StartPoint().Row) + 1
	endRow := int(node.EndPoint().Row) + 1

	qualified := name
	if className != "" {
		qualified = className + "." + name
	}
	if pkg != "" {
		qualified = pkg + "." + qualified
	}

	exported := false
	for i := 0; i < int(node.NamedChildCount()); i++ {
		ch := node.NamedChild(i)
		if ch.Type() == "modifiers" && strings.Contains(ch.Content(content), "public") {
			exported = true
			break
		}
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
		Receiver:      className,
		Exported:      exported,
		Language:      "java",
		Body:          body,
		TokenCount:    EstimateTokens(body),
	}
}

func (j *JavaExtractor) extractImport(node *sitter.Node, content []byte) []ImportReference {
	var results []ImportReference
	text := node.Content(content)
	text = strings.TrimPrefix(text, "import ")
	text = strings.TrimPrefix(text, "static ")
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)
	if text != "" {
		results = append(results, ImportReference{
			Path: text,
		})
	}
	return results
}

func (j *JavaExtractor) extractCalls(root *sitter.Node, content []byte, filePath string) []CallReference {
	var calls []CallReference

	var traverse func(node *sitter.Node, currentCaller string)
	traverse = func(n *sitter.Node, currentCaller string) {
		if n == nil {
			return
		}

		newCaller := currentCaller
		if n.Type() == "method_declaration" || n.Type() == "constructor_declaration" {
			nameNode := n.ChildByFieldName("name")
			if nameNode != nil {
				newCaller = nameNode.Content(content)
			}
		}

		if n.Type() == "method_invocation" {
			nameNode := n.ChildByFieldName("name")
			objNode := n.ChildByFieldName("object")

			calleeName := ""
			receiver := ""
			isMethod := false

			if nameNode != nil {
				calleeName = nameNode.Content(content)
			}
			if objNode != nil {
				receiver = objNode.Content(content)
				isMethod = true
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

		for i := 0; i < int(n.NamedChildCount()); i++ {
			traverse(n.NamedChild(i), newCaller)
		}
	}

	traverse(root, "")
	return calls
}

func (j *JavaExtractor) extractDocstring(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	var comments []string
	for prev != nil {
		if prev.Type() == "block_comment" || prev.Type() == "line_comment" {
			comments = append([]string{prev.Content(content)}, comments...)
		} else if strings.TrimSpace(prev.Content(content)) != "" {
			break
		}
		prev = prev.PrevSibling()
	}
	return strings.Join(comments, "\n")
}
