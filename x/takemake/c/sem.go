package c

func splice(src []byte) []byte {
	out := make([]byte, 0, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] == '\\' && i+1 < len(src) && (src[i+1] == '\n' || src[i+1] == '\r') {
			i++
			if src[i] == '\r' && i+1 < len(src) && src[i+1] == '\n' {
				i++
			}
			continue
		}
		out = append(out, src[i])
	}
	return out
}

func lineAt(src []byte, at int) int {
	if at < 0 {
		at = 0
	}
	if at > len(src) {
		at = len(src)
	}
	line := 1
	for i := 0; i < at; i++ {
		if src[i] == '\n' {
			line++
		}
	}
	return line
}

func sameAggregate(typ, raw *Type) bool {
	if typ == nil || raw == nil {
		return false
	}
	if typ == raw {
		return true
	}
	return typ.Of == raw
}

func qualify(t *Type, isConst, isVolatile bool) *Type {
	if t == nil || (!isConst && !isVolatile) {
		return t
	}
	n := *t
	n.Const = t.Const || isConst
	n.Volatile = t.Volatile || isVolatile
	n.Of = t
	n.Fields = nil
	n.Enums = nil
	n.Defined = false
	n.Inline = false
	n.home = nil
	n.laying = false
	layout(&n)
	return &n
}

func isAggregate(t *Type) bool {
	return t != nil && (t.Kind == KindStruct || t.Kind == KindUnion || t.Kind == KindEnum)
}

// detachDecl gives a declarator its own type nodes. A comma list is
// printed as separate declarations, and a later take builds a fresh
// node for each one. The shared tag of a struct, union, or enum stays
// one node so a pointer back to it still lines up. A typedef use keeps
// Of on the typedef's type.
func detachDecl(t *Type) *Type {
	return cloneType(t, map[*Type]*Type{})
}

func cloneType(t *Type, seen map[*Type]*Type) *Type {
	if t == nil {
		return nil
	}
	if n, ok := seen[t]; ok {
		return n
	}
	if t.Use == "" && t.Of == nil && t.Tag != "" && isAggregate(t) {
		return t
	}
	n := *t
	n.home = nil
	n.laying = false
	seen[t] = &n
	if t.Use != "" {
		return &n
	}
	n.Elem = cloneType(t.Elem, seen)
	n.Of = cloneType(t.Of, seen)
	if len(t.Fields) > 0 {
		n.Fields = make([]*Field, len(t.Fields))
		for i, field := range t.Fields {
			if field == nil {
				continue
			}
			cp := *field
			cp.Type = cloneType(field.Type, seen)
			n.Fields[i] = &cp
		}
	}
	if len(t.Params) > 0 {
		n.Params = make([]*Param, len(t.Params))
		for i, param := range t.Params {
			if param == nil {
				continue
			}
			cp := *param
			cp.Type = cloneType(param.Type, seen)
			n.Params[i] = &cp
		}
	}
	return &n
}

func evalExpr(e *Expr, names map[string]int64) (int64, bool) {
	if e == nil {
		return 0, false
	}
	switch e.Kind {
	case ExprInt:
		return e.Int, true
	case ExprChar:
		return e.Int, true
	case ExprIdent:
		v, ok := names[e.Text]
		return v, ok
	case ExprUnary:
		v, ok := evalExpr(e.Left, names)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case OpNeg:
			return -v, true
		case OpPos:
			return v, true
		case OpBitNot:
			return ^v, true
		case OpNot:
			if v == 0 {
				return 1, true
			}
			return 0, true
		default:
			return 0, false
		}
	case ExprBinary:
		l, ok := evalExpr(e.Left, names)
		if !ok {
			return 0, false
		}
		r, ok := evalExpr(e.Right, names)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case OpAdd:
			return l + r, true
		case OpSub:
			return l - r, true
		case OpMul:
			return l * r, true
		case OpDiv:
			if r == 0 {
				return 0, false
			}
			return l / r, true
		case OpMod:
			if r == 0 {
				return 0, false
			}
			return l % r, true
		case OpEq:
			return boolInt(l == r), true
		case OpNe:
			return boolInt(l != r), true
		case OpLt:
			return boolInt(l < r), true
		case OpGt:
			return boolInt(l > r), true
		case OpLe:
			return boolInt(l <= r), true
		case OpGe:
			return boolInt(l >= r), true
		case OpAnd:
			return boolInt(l != 0 && r != 0), true
		case OpOr:
			return boolInt(l != 0 || r != 0), true
		case OpBitAnd:
			return l & r, true
		case OpBitOr:
			return l | r, true
		case OpBitXor:
			return l ^ r, true
		case OpShl:
			if r < 0 || r >= 64 {
				return 0, false
			}
			return l << uint64(r), true
		case OpShr:
			if r < 0 || r >= 64 {
				return 0, false
			}
			return l >> uint64(r), true
		default:
			return 0, false
		}
	case ExprSizeof:
		if e.Type == nil {
			return 0, false
		}
		layout(e.Type)
		return int64(e.Type.Size), true
	default:
		return 0, false
	}
}

func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func decodeEscape(s []byte) (rune, int, bool) {
	if len(s) < 2 || s[0] != '\\' {
		return 0, 0, false
	}
	switch s[1] {
	case 'n':
		return '\n', 2, true
	case 'r':
		return '\r', 2, true
	case 't':
		return '\t', 2, true
	case 'a':
		return '\a', 2, true
	case 'b':
		return '\b', 2, true
	case 'f':
		return '\f', 2, true
	case 'v':
		return '\v', 2, true
	case '\\', '\'', '"', '?':
		return rune(s[1]), 2, true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(0)
		i := 1
		for i < len(s) && i <= 3 && s[i] >= '0' && s[i] <= '7' {
			v = v*8 + rune(s[i]-'0')
			i++
		}
		return v, i, true
	case 'x':
		v := rune(0)
		i := 2
		if i >= len(s) || !isHex(s[i]) {
			return 0, 0, false
		}
		for i < len(s) && isHex(s[i]) {
			v = v*16 + rune(hexVal(s[i]))
			i++
		}
		return v, i, true
	default:
		return 0, 0, false
	}
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return c - 'A' + 10
	}
}
