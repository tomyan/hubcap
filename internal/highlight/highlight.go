package highlight

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

// Span represents a styled segment of text.
type Span struct {
	Text string
	Kind string // "plain", "keyword", "string", "number", "property", "comment", "operator"
}

var parser *sitter.Parser

func init() {
	parser = sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())
}

// Highlight parses a JavaScript expression and returns colored spans.
func Highlight(src string) []Span {
	if src == "" {
		return nil
	}

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(src))
	if err != nil {
		return []Span{{Text: src, Kind: "plain"}}
	}
	defer tree.Close()

	var spans []Span
	pos := 0
	walkLeaves(tree.RootNode(), src, &spans, &pos)

	// Emit any trailing text not covered by leaves.
	if pos < len(src) {
		spans = append(spans, Span{Text: src[pos:], Kind: "plain"})
	}

	return spans
}

func walkLeaves(node *sitter.Node, src string, spans *[]Span, pos *int) {
	if node.ChildCount() == 0 {
		start := int(node.StartByte())
		end := int(node.EndByte())

		// Fill gap before this leaf.
		if start > *pos {
			*spans = append(*spans, Span{Text: src[*pos:start], Kind: "plain"})
		}

		if end > start && end <= len(src) {
			kind := classifyNode(node)
			*spans = append(*spans, Span{Text: src[start:end], Kind: kind})
		}
		if end > *pos {
			*pos = end
		}
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		walkLeaves(node.Child(i), src, spans, pos)
	}
}

func classifyNode(node *sitter.Node) string {
	typ := node.Type()

	switch typ {
	// Keywords
	case "const", "let", "var", "function", "return", "if", "else",
		"for", "while", "do", "switch", "case", "break", "continue",
		"new", "delete", "typeof", "instanceof", "in", "of",
		"try", "catch", "finally", "throw",
		"class", "extends", "import", "export", "default", "from",
		"async", "await", "yield",
		"this", "super",
		"true", "false", "null", "undefined", "void":
		return "keyword"

	// Strings
	case "string", "template_string", "string_fragment", "template_substitution":
		return "string"

	// Numbers
	case "number":
		return "number"

	// Properties
	case "property_identifier", "shorthand_property_identifier",
		"shorthand_property_identifier_pattern":
		return "property"

	// Comments
	case "comment":
		return "comment"

	// Operators
	case "+", "-", "*", "/", "%", "**",
		"=", "+=", "-=", "*=", "/=", "%=",
		"==", "!=", "===", "!==",
		"<", ">", "<=", ">=",
		"&&", "||", "!",
		"&", "|", "^", "~", "<<", ">>", ">>>",
		"?", ":", "??",
		"=>", "...",
		"++", "--":
		return "operator"

	default:
		return "plain"
	}
}
