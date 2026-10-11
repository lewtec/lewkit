package clib

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/generate/header"
)

type regKind int

const (
	regGPR regKind = iota
	regXMM
)

type valKind int

const (
	vkVoid valKind = iota
	vkBool
	vkSInt
	vkUInt
	vkF32
	vkF64
	vkPtr
)

type val struct {
	kind   valKind
	goType string
	reg    regKind
	size   int
}

type fn struct {
	cName  string
	goName string
	call   string
	id     int
	ret    val
	params []param
}

type param struct {
	name string
	val  val
	at   loc
}

func classify(funcs []header.Func, m machine) ([]fn, error) {
	used := map[string]int{}
	out := make([]fn, 0, len(funcs))
	for _, raw := range funcs {
		if !cNameOK(raw.Name) {
			return nil, fmt.Errorf("%w: %s", ErrSymbol, raw.Name)
		}
		item, err := classifyFunc(raw, m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", raw.Name, err)
		}
		base := exportName(raw.Name)
		n := used[base]
		used[base] = n + 1
		item.goName = base
		if n > 0 {
			item.goName = fmt.Sprintf("%s%d", base, n+1)
		}
		item.call = "call" + item.goName
		item.id = len(out)
		out = append(out, item)
	}
	return out, nil
}

func classifyFunc(raw header.Func, m machine) (fn, error) {
	ret, err := classifyType(raw.Ret, true)
	if err != nil {
		return fn{}, err
	}
	params := make([]param, len(raw.Params))
	seen := map[string]bool{}
	for i, p := range raw.Params {
		v, err := classifyType(p.Type, false)
		if err != nil {
			return fn{}, err
		}
		if v.kind == vkVoid {
			return fn{}, ErrType
		}
		params[i] = param{name: paramName(p.Name, i, seen), val: v}
	}
	if err := place(m, params); err != nil {
		return fn{}, err
	}
	return fn{cName: raw.Name, ret: ret, params: params}, nil
}

func classifyType(t *header.Type, asReturn bool) (val, error) {
	c, err := concrete(t)
	if err != nil {
		return val{}, err
	}
	switch header.ShapeOf(c) {
	case header.ShapeVoid:
		if !asReturn {
			return val{}, ErrType
		}
		return val{kind: vkVoid}, nil
	case header.ShapeBool:
		return val{kind: vkBool, goType: "bool", size: c.Size}, nil
	case header.ShapeSInt:
		return intVal(c, true)
	case header.ShapeUInt:
		return intVal(c, false)
	case header.ShapeEnum:
		return intVal(c, true)
	case header.ShapeFloat:
		switch c.Size {
		case 4:
			return val{kind: vkF32, goType: "float32", reg: regXMM, size: 4}, nil
		case 8:
			return val{kind: vkF64, goType: "float64", reg: regXMM, size: 8}, nil
		default:
			return val{}, ErrType
		}
	case header.ShapePtr:
		v, err := ptrVal(c.Elem)
		if err != nil {
			return val{}, err
		}
		v.size = c.Size
		return v, nil
	default:
		return val{}, ErrType
	}
}

func concrete(t *header.Type) (*header.Type, error) {
	for i := 0; i < 8; i++ {
		if t == nil {
			return nil, ErrType
		}
		if header.ShapeOf(t) != header.ShapeAlias {
			return t, nil
		}
		t = t.Elem
	}
	return nil, ErrType
}

func intVal(t *header.Type, signed bool) (val, error) {
	if t.Spell == "intptr_t" || t.Spell == "uintptr_t" {
		return val{kind: vkUInt, goType: "uintptr"}, nil
	}
	var g string
	switch t.Size {
	case 1:
		if signed {
			g = "int8"
		} else {
			g = "byte"
		}
	case 2:
		if signed {
			g = "int16"
		} else {
			g = "uint16"
		}
	case 4:
		if signed {
			g = "int32"
		} else {
			g = "uint32"
		}
	case 8:
		if signed {
			g = "int64"
		} else {
			g = "uint64"
		}
	default:
		return val{}, ErrType
	}
	k := vkUInt
	if signed {
		k = vkSInt
	}
	return val{kind: k, goType: g, size: t.Size}, nil
}

func ptrVal(elem *header.Type) (val, error) {
	c, err := concrete(elem)
	if err != nil || c == nil {
		return val{kind: vkPtr, goType: "unsafe.Pointer"}, nil
	}
	switch header.ShapeOf(c) {
	case header.ShapeVoid, header.ShapeFunc, header.ShapeStruct, header.ShapeUnion, header.ShapeArray:
		return val{kind: vkPtr, goType: "unsafe.Pointer"}, nil
	default:
		inner, err := classifyType(c, false)
		if err != nil || inner.goType == "" {
			return val{kind: vkPtr, goType: "unsafe.Pointer"}, nil
		}
		return val{kind: vkPtr, goType: "*" + inner.goType}, nil
	}
}

func cNameOK(name string) bool {
	if name == "" || name == "main" || name == "_start" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if i == 0 {
			if c != '_' && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
				return false
			}
			continue
		}
		if c != '_' && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func paramName(name string, i int, seen map[string]bool) string {
	if !token.IsIdentifier(name) || token.IsKeyword(name) {
		name = fmt.Sprintf("p%d", i)
	}
	if seen[name] {
		name = fmt.Sprintf("p%d", i)
	}
	seen[name] = true
	return name
}

func exportName(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	if len(parts) == 1 {
		b.WriteString(capitalize(parts[0]))
	} else {
		for _, part := range parts {
			if part == "" {
				continue
			}
			b.WriteString(capitalize(part))
		}
	}
	s := b.String()
	if s == "" {
		s = "Anon"
	}
	r, _ := utf8.DecodeRuneInString(s)
	if r >= '0' && r <= '9' {
		s = "N" + s
	}
	if token.IsKeyword(s) {
		s += "X"
	}
	if !token.IsIdentifier(s) {
		s = "Name"
	}
	return s
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	if unicode.IsLower(r) {
		return string(unicode.ToUpper(r)) + s[size:]
	}
	return s
}
