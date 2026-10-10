package header

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/treesitter"
)

func (p *parser) evalSnippet(text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%w: empty", errExpr)
	}
	src := "enum { X = (" + text + ") };"
	tree, err := treesitter.Parse(p.ctx, "c", []byte(src))
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errExpr, text)
	}
	if err := tree.Parsed(); err != nil {
		return 0, fmt.Errorf("%w: %s", errExpr, text)
	}
	val := findNode(tree.RootNode(), "enumerator")
	if val == nil {
		return 0, fmt.Errorf("%w: %s", errExpr, text)
	}
	got := childField(val, "value")
	if got == nil || got.IsNull() {
		return 0, fmt.Errorf("%w: %s", errExpr, text)
	}
	return p.eval(src, got)
}

func findNode(n treesitter.Node, typ string) treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	if n.Type() == typ {
		return n
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		if hit := findNode(n.Child(i), typ); hit != nil {
			return hit
		}
	}
	return nil
}

func (p *parser) defined(name string) bool {
	_, ok := p.macros[name]
	return ok
}

func (p *parser) macroValue(name string) (int64, bool, error) {
	m, ok := p.macros[name]
	if !ok {
		return 0, false, nil
	}
	if m.fn || strings.TrimSpace(m.text) == "" {
		return 0, true, fmt.Errorf("%w: %s", errExpr, name)
	}
	if m.known {
		return m.value, true, nil
	}
	if p.expanding[name] {
		return 0, true, fmt.Errorf("%w: %s", errExpr, name)
	}
	p.expanding[name] = true
	v, err := p.evalSnippet(m.text)
	delete(p.expanding, name)
	if err != nil {
		return 0, true, err
	}
	m.known = true
	m.value = v
	return v, true, nil
}

func (p *parser) eval(src string, n treesitter.Node) (int64, error) {
	if n == nil || n.IsNull() {
		return 0, errExpr
	}
	if n.IsError() || n.Type() == "ERROR" {
		return 0, errSyntax
	}
	switch n.Type() {
	case "number_literal":
		return parseNumber(nodeText(src, n))
	case "char_literal":
		return parseChar(nodeText(src, n))
	case "identifier", "type_identifier":
		name := nodeText(src, n)
		if v, ok, err := p.macroValue(name); ok {
			if err != nil && p.inIf {
				return 0, nil
			}
			return v, err
		}
		if p.inIf {
			return 0, nil
		}
		if v, ok := p.enums[name]; ok {
			return v, nil
		}
		return 0, fmt.Errorf("%w: %s", errExpr, name)
	case "preproc_defined":
		id := ""
		for i := uint32(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c != nil && !c.IsNull() && (c.Type() == "identifier" || c.Type() == "type_identifier") {
				id = nodeText(src, c)
				break
			}
		}
		if p.defined(id) {
			return 1, nil
		}
		return 0, nil
	case "unary_expression":
		op := nodeText(src, childField(n, "operator"))
		arg, err := p.eval(src, childField(n, "argument"))
		if err != nil {
			return 0, err
		}
		switch op {
		case "!":
			if arg == 0 {
				return 1, nil
			}
			return 0, nil
		case "~":
			return ^arg, nil
		case "-":
			return -arg, nil
		case "+":
			return arg, nil
		default:
			return 0, fmt.Errorf("%w: unary %s", errExpr, op)
		}
	case "binary_expression":
		left, err := p.eval(src, childField(n, "left"))
		if err != nil {
			return 0, err
		}
		right, err := p.eval(src, childField(n, "right"))
		if err != nil {
			return 0, err
		}
		return foldBinary(nodeText(src, childField(n, "operator")), left, right)
	case "parenthesized_expression":
		for i := uint32(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c == nil || c.IsNull() || !c.IsNamed() {
				continue
			}
			return p.eval(src, c)
		}
		return 0, errExpr
	case "sizeof_expression":
		if t := childField(n, "type"); t != nil && !t.IsNull() {
			return p.sizeofType(src, t)
		}
		for i := uint32(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c != nil && !c.IsNull() && c.Type() == "type_descriptor" {
				return p.sizeofType(src, c)
			}
		}
		return 0, fmt.Errorf("%w: sizeof", errExpr)
	case "type_descriptor":
		return p.sizeofType(src, n)
	case "preproc_arg":
		return p.evalSnippet(nodeText(src, n))
	case "conditional_expression":
		return 0, fmt.Errorf("%w: conditional", errExpr)
	default:
		return 0, fmt.Errorf("%w: %s", errExpr, n.Type())
	}
}

func (p *parser) sizeofType(src string, n treesitter.Node) (int64, error) {
	inner := n
	if n.Type() == "type_descriptor" {
		if t := childField(n, "type"); t != nil && !t.IsNull() {
			inner = t
		}
	}
	typ, err := p.typeOf(src, inner)
	if err != nil {
		return 0, err
	}
	if err := p.layoutOne(typ); err != nil {
		return 0, err
	}
	return int64(typ.Size), nil
}

func foldBinary(op string, left, right int64) (int64, error) {
	switch op {
	case "+":
		return left + right, nil
	case "-":
		return left - right, nil
	case "*":
		return left * right, nil
	case "/":
		if right == 0 {
			return 0, fmt.Errorf("%w: division", errExpr)
		}
		return left / right, nil
	case "%":
		if right == 0 {
			return 0, fmt.Errorf("%w: modulo", errExpr)
		}
		return left % right, nil
	case "&":
		return left & right, nil
	case "|":
		return left | right, nil
	case "^":
		return left ^ right, nil
	case "<<":
		if right < 0 || right >= 64 {
			return 0, fmt.Errorf("%w: shift", errExpr)
		}
		return left << uint64(right), nil
	case ">>":
		if right < 0 || right >= 64 {
			return 0, fmt.Errorf("%w: shift", errExpr)
		}
		return left >> uint64(right), nil
	case "&&":
		if left != 0 && right != 0 {
			return 1, nil
		}
		return 0, nil
	case "||":
		if left != 0 || right != 0 {
			return 1, nil
		}
		return 0, nil
	case "==":
		if left == right {
			return 1, nil
		}
		return 0, nil
	case "!=":
		if left != right {
			return 1, nil
		}
		return 0, nil
	case "<":
		if left < right {
			return 1, nil
		}
		return 0, nil
	case ">":
		if left > right {
			return 1, nil
		}
		return 0, nil
	case "<=":
		if left <= right {
			return 1, nil
		}
		return 0, nil
	case ">=":
		if left >= right {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("%w: %s", errExpr, op)
	}
}

func parseNumber(raw string) (int64, error) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "'", "")
	s = strings.TrimRight(s, "uUlL")
	if s == "" {
		return 0, fmt.Errorf("%w: %s", errExpr, raw)
	}
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errExpr, raw)
	}
	return v, nil
}

func parseChar(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	r, _, _, err := strconv.UnquoteChar(s, '\'')
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errExpr, raw)
	}
	return int64(r), nil
}
