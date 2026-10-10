package header

import "github.com/lewtec/lewkit/x/driver/treesitter"

func nodeText(src string, n treesitter.Node) string {
	if n == nil || n.IsNull() {
		return ""
	}
	start, end := int(n.StartByte()), int(n.EndByte())
	if start < 0 || end > len(src) || start > end {
		return ""
	}
	return src[start:end]
}

func childField(n treesitter.Node, name string) treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) == name {
			return n.Child(i)
		}
	}
	return nil
}

func childFields(n treesitter.Node, name string) []treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	var out []treesitter.Node
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) == name {
			out = append(out, n.Child(i))
		}
	}
	return out
}

func lineAt(src string, n treesitter.Node) int {
	line := 1
	if n == nil || n.IsNull() {
		return line
	}
	end := int(n.StartByte())
	if end > len(src) {
		end = len(src)
	}
	for i := 0; i < end; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}

func isDeclarator(typ string) bool {
	switch typ {
	case "identifier", "field_identifier", "type_identifier", "primitive_type",
		"pointer_declarator", "array_declarator", "function_declarator", "parenthesized_declarator",
		"abstract_pointer_declarator", "abstract_array_declarator", "abstract_function_declarator",
		"abstract_parenthesized_declarator", "init_declarator":
		return true
	default:
		return false
	}
}

// innerDecl is the declarator nested inside n.
// Parenthesized declarators often leave that child untagged.
func innerDecl(n treesitter.Node) treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	if d := childField(n, "declarator"); d != nil && !d.IsNull() {
		return d
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		if isDeclarator(c.Type()) {
			return c
		}
	}
	return nil
}
