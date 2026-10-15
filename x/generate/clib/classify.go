package clib

import (
	"fmt"
	"go/token"

	"github.com/lewtec/lewkit/x/generate/header"
)

//lewkit:enum
type regKind int

const (
	regGPR regKind = iota
	regXMM
)

//lewkit:enum
type valKind int

const (
	vkVoid valKind = iota
	vkBool
	vkSInt
	vkUInt
	vkF32
	vkF64
	vkPtr
	vkStruct
)

type val struct {
	kind   valKind
	goType string
	reg    regKind
	size   int
	// words is the number of integer eightbytes in a by-value struct.
	words int
	def   *sdef
}

type sfield struct {
	name   string
	goType string
	offset int
	size   int
	kind   valKind
}

// sdef is one generated Go struct. Fields use C offsets.
type sdef struct {
	name   string
	size   int
	fields []sfield
}

type class struct {
	defs  map[*header.Type]*sdef
	order []*sdef
	names map[string]int
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
	ats  []loc
}

func classify(funcs []header.Func, m machine) ([]fn, []*sdef, error) {
	c := &class{defs: map[*header.Type]*sdef{}, names: map[string]int{}}
	used := map[string]int{}
	out := make([]fn, 0, len(funcs))
	for _, raw := range funcs {
		if !cNameOK(raw.Name) {
			return nil, nil, fmt.Errorf("%w: %s", ErrSymbol, raw.Name)
		}
		item, err := c.classifyFunc(raw, m)
		if err != nil {
			return nil, nil, fmt.Errorf("%w%s: %w", ErrClassify, raw.Name, err)
		}
		base := header.ExportName(raw.Name)
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
	return out, c.order, nil
}

func (c *class) classifyFunc(raw header.Func, m machine) (fn, error) {
	ret, err := c.classifyType(raw.Ret, true)
	if err != nil {
		return fn{}, err
	}
	params := make([]param, len(raw.Params))
	seen := map[string]bool{}
	for i, p := range raw.Params {
		v, err := c.classifyType(p.Type, false)
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

func (c *class) classifyType(t *header.Type, asReturn bool) (val, error) {
	ct, err := concrete(t)
	if err != nil {
		return val{}, err
	}
	switch header.ShapeOf(ct) {
	case header.ShapeVoid:
		if !asReturn {
			return val{}, ErrType
		}
		return val{kind: vkVoid}, nil
	case header.ShapeBool:
		return val{kind: vkBool, goType: "bool", size: ct.Size}, nil
	case header.ShapeSInt:
		return intVal(ct, true)
	case header.ShapeUInt:
		return intVal(ct, false)
	case header.ShapeEnum:
		return intVal(ct, true)
	case header.ShapeFloat:
		if ct.Spell == "long double" {
			return val{}, ErrType
		}
		switch ct.Size {
		case 4:
			return val{kind: vkF32, goType: "float32", reg: regXMM, size: 4}, nil
		case 8:
			return val{kind: vkF64, goType: "float64", reg: regXMM, size: 8}, nil
		default:
			return val{}, ErrType
		}
	case header.ShapePtr:
		v, err := c.ptrVal(ct.Elem)
		if err != nil {
			return val{}, err
		}
		v.size = ct.Size
		return v, nil
	case header.ShapeStruct:
		return c.structByValue(ct)
	default:
		return val{}, ErrType
	}
}

func (c *class) structByValue(t *header.Type) (val, error) {
	d, err := c.define(t)
	if err != nil {
		return val{}, err
	}
	if t.Packed || t.Size <= 0 || t.Size > 16 {
		return val{}, ErrType
	}
	for _, f := range d.fields {
		if f.kind == vkF32 || f.kind == vkF64 || f.kind == vkStruct {
			return val{}, ErrType
		}
		if f.size > 0 && f.offset/8 != (f.offset+f.size-1)/8 {
			return val{}, ErrType
		}
	}
	return val{kind: vkStruct, goType: d.name, size: t.Size, words: (t.Size + 7) / 8, def: d}, nil
}

func (c *class) define(t *header.Type) (*sdef, error) {
	if t == nil || t.Incomplete || header.ShapeOf(t) != header.ShapeStruct {
		return nil, ErrType
	}
	if d, ok := c.defs[t]; ok {
		return d, nil
	}
	d := &sdef{name: c.unique(structBase(t)), size: t.Size}
	c.defs[t] = d
	c.order = append(c.order, d)
	seen := map[string]bool{}
	for i, f := range t.Fields {
		if f == nil || f.IsBit || f.Type == nil || flexible(f.Type) {
			return nil, ErrType
		}
		fv, err := c.classifyType(f.Type, false)
		if err != nil {
			return nil, err
		}
		if fv.kind == vkStruct {
			return nil, ErrType
		}
		name := f.Go
		if name == "" {
			name = f.Name
		}
		name = header.ExportName(name)
		if seen[name] {
			name = fmt.Sprintf("F%d", i)
		}
		seen[name] = true
		d.fields = append(d.fields, sfield{
			name: name, goType: fv.goType, offset: f.Offset, size: f.CSize, kind: fv.kind,
		})
	}
	if len(d.fields) == 0 {
		return nil, ErrType
	}
	return d, nil
}

func structBase(t *header.Type) string {
	base := t.Go
	if base == "" {
		base = t.Spell
	}
	if base == "" {
		base = t.Tag
	}
	if base == "" || base == "struct" || base == "union" || base == "enum" {
		base = "Anon"
	}
	return header.ExportName(base)
}

func flexible(t *header.Type) bool {
	return t != nil && header.ShapeOf(t) == header.ShapeArray && t.Len < 0
}

func (c *class) unique(name string) string {
	n := c.names[name]
	c.names[name] = n + 1
	if n == 0 {
		return name
	}
	return fmt.Sprintf("%s%d", name, n+1)
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

func (c *class) ptrVal(elem *header.Type) (val, error) {
	ct, err := concrete(elem)
	if err != nil || ct == nil {
		return val{kind: vkPtr, goType: "unsafe.Pointer"}, nil
	}
	if header.ShapeOf(ct) == header.ShapeStruct {
		d, err := c.define(ct)
		if err != nil {
			return val{}, err
		}
		return val{kind: vkPtr, goType: "*" + d.name}, nil
	}
	switch header.ShapeOf(ct) {
	case header.ShapeVoid, header.ShapeFunc, header.ShapeUnion, header.ShapeArray:
		return val{kind: vkPtr, goType: "unsafe.Pointer"}, nil
	default:
		inner, err := c.classifyType(ct, false)
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
