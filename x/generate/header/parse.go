package header

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lewtec/leaven-tree-sitter/grammar/c"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	_ "github.com/lewtec/lewkit/x/driver/treesitter/leaven"
)

// Options selects include paths, macros, and the C layout.
// A zero Target uses Host.
type Options struct {
	Include []string
	Define  []string
	Target  Target
}

type macro struct {
	text   string
	fn     bool
	header bool
	known  bool
	value  int64
}

type parser struct {
	ctx       context.Context
	target    Target
	include   []string
	macros    map[string]*macro
	typedefs  map[string]*Type
	tags      map[string]*Type
	enums     map[string]int64
	visited   map[string]bool
	expanding map[string]bool
	unit      *Unit
	inIf      bool

	tyVoid       *Type
	tyBool       *Type
	tyChar       *Type
	tySChar      *Type
	tyUChar      *Type
	tyShort      *Type
	tyUShort     *Type
	tyInt        *Type
	tyUInt       *Type
	tyLong       *Type
	tyULong      *Type
	tyLongLong   *Type
	tyULongLong  *Type
	tyFloat      *Type
	tyDouble     *Type
	tyLongDouble *Type
}

// Parse reads a C header through the tree-sitter driver and records
// the types, macros, and functions a purego binding can use.
func Parse(ctx context.Context, file string, opt Options) (*Unit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if file == "" {
		return nil, errHeader
	}
	target := opt.Target
	if target.ID == "" {
		target = Host()
	}
	p := newParser(ctx, target, opt.Include)
	p.seed()
	for _, spec := range opt.Define {
		if err := p.userDefine(spec); err != nil {
			return nil, err
		}
	}
	if err := p.parseFile(file); err != nil {
		return nil, err
	}
	if err := p.finish(); err != nil {
		return nil, err
	}
	return p.unit, nil
}

func newParser(ctx context.Context, target Target, include []string) *parser {
	return &parser{
		ctx:       ctx,
		target:    target,
		include:   append([]string(nil), include...),
		macros:    map[string]*macro{},
		typedefs:  map[string]*Type{},
		tags:      map[string]*Type{},
		enums:     map[string]int64{},
		visited:   map[string]bool{},
		expanding: map[string]bool{},
		unit:      &Unit{Target: target},
	}
}

func (p *parser) scalar(k kind, size, align int, spell string) *Type {
	if align < 1 {
		align = 1
	}
	return &Type{Kind: k, Size: size, Align: align, Spell: spell, laid: true, natKnown: true, nat: true}
}

func (p *parser) seed() {
	t := p.target
	llAlign := 8
	doubleAlign := 8
	if t.I386 {
		llAlign = 4
		doubleAlign = 4
	}
	p.tyVoid = p.scalar(kindVoid, 0, 1, "void")
	p.tyBool = p.scalar(kindBool, 1, 1, "_Bool")
	p.tyChar = p.scalar(kindUInt, 1, 1, "char")
	p.tySChar = p.scalar(kindSInt, 1, 1, "signed char")
	p.tyUChar = p.scalar(kindUInt, 1, 1, "unsigned char")
	p.tyShort = p.scalar(kindSInt, 2, 2, "short")
	p.tyUShort = p.scalar(kindUInt, 2, 2, "unsigned short")
	p.tyInt = p.scalar(kindSInt, t.Int, t.Int, "int")
	p.tyUInt = p.scalar(kindUInt, t.Int, t.Int, "unsigned int")
	p.tyLong = p.scalar(kindSInt, t.Long, t.Long, "long")
	p.tyULong = p.scalar(kindUInt, t.Long, t.Long, "unsigned long")
	p.tyLongLong = p.scalar(kindSInt, 8, llAlign, "long long")
	p.tyULongLong = p.scalar(kindUInt, 8, llAlign, "unsigned long long")
	p.tyFloat = p.scalar(kindFloat, 4, 4, "float")
	p.tyDouble = p.scalar(kindFloat, 8, doubleAlign, "double")
	p.tyLongDouble = p.scalar(kindFloat, t.LongDouble, t.LongAlign, "long double")

	p.typedefs["int8_t"] = p.scalar(kindSInt, 1, 1, "int8_t")
	p.typedefs["uint8_t"] = p.scalar(kindUInt, 1, 1, "uint8_t")
	p.typedefs["int16_t"] = p.scalar(kindSInt, 2, 2, "int16_t")
	p.typedefs["uint16_t"] = p.scalar(kindUInt, 2, 2, "uint16_t")
	p.typedefs["int32_t"] = p.scalar(kindSInt, 4, 4, "int32_t")
	p.typedefs["uint32_t"] = p.scalar(kindUInt, 4, 4, "uint32_t")
	p.typedefs["int64_t"] = p.scalar(kindSInt, 8, llAlign, "int64_t")
	p.typedefs["uint64_t"] = p.scalar(kindUInt, 8, llAlign, "uint64_t")
	p.typedefs["intptr_t"] = p.scalar(kindSInt, t.Ptr, t.Ptr, "intptr_t")
	p.typedefs["uintptr_t"] = p.scalar(kindUInt, t.Ptr, t.Ptr, "uintptr_t")
	p.typedefs["intmax_t"] = p.scalar(kindSInt, 8, llAlign, "intmax_t")
	p.typedefs["uintmax_t"] = p.scalar(kindUInt, 8, llAlign, "uintmax_t")
	p.typedefs["size_t"] = p.tyULong
	p.typedefs["ssize_t"] = p.tyLong
	p.typedefs["ptrdiff_t"] = p.tyLong
	if t.WcharSign {
		p.typedefs["wchar_t"] = p.scalar(kindSInt, t.Wchar, t.Wchar, "wchar_t")
	} else {
		p.typedefs["wchar_t"] = p.scalar(kindUInt, t.Wchar, t.Wchar, "wchar_t")
	}
	p.typedefs["char16_t"] = p.scalar(kindUInt, 2, 2, "char16_t")
	p.typedefs["char32_t"] = p.scalar(kindUInt, 4, 4, "char32_t")

	p.define("__STDC__", "1", false)
	if t.Ptr == 8 && t.Long == 8 {
		p.define("__LP64__", "1", false)
	}
	if t.Linux {
		p.define("__linux__", "1", false)
		p.define("__unix__", "1", false)
	}
	if t.Darwin {
		p.define("__APPLE__", "1", false)
		p.define("__unix__", "1", false)
	}
	if t.Windows {
		p.define("_WIN32", "1", false)
		p.define("_WIN64", "1", false)
	}
	if t.AMD64 {
		p.define("__x86_64__", "1", false)
		p.define("__amd64__", "1", false)
	}
	if t.ARM64 {
		p.define("__aarch64__", "1", false)
	}
	if t.I386 {
		p.define("__i386__", "1", false)
	}
	if t.ARM {
		p.define("__arm__", "1", false)
	}
	p.define("__SIZEOF_POINTER__", fmt.Sprint(t.Ptr), false)
	p.define("__SIZEOF_INT__", fmt.Sprint(t.Int), false)
	p.define("__SIZEOF_LONG__", fmt.Sprint(t.Long), false)
	p.define("__SIZEOF_LONG_DOUBLE__", fmt.Sprint(t.LongDouble), false)
}

func (p *parser) define(name, text string, header bool) {
	m := &macro{text: strings.TrimSpace(text), header: header}
	p.macros[name] = m
	if m.text == "" {
		return
	}
	if v, err := p.evalSnippet(m.text); err == nil {
		m.known = true
		m.value = v
	}
}

func (p *parser) userDefine(spec string) error {
	name, value, ok := strings.Cut(spec, "=")
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: define", errExpr)
	}
	if !ok {
		value = "1"
	}
	p.define(name, value, false)
	return nil
}

func (p *parser) parseFile(path string) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if p.visited[abs] {
		return nil
	}
	p.visited[abs] = true
	buf, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	src := string(buf)
	tree, err := treesitter.Parse(p.ctx, "c", buf)
	if err != nil {
		return fmt.Errorf("%s: %w", abs, err)
	}
	if err := tree.Parsed(); err != nil {
		return fmt.Errorf("%s: %w", abs, errSyntax)
	}
	root := tree.RootNode()
	if root == nil || root.IsNull() {
		return nil
	}
	return p.walkBody(abs, src, root)
}

func (p *parser) walkBody(path, src string, n treesitter.Node) error {
	for i := uint32(0); i < n.ChildCount(); i++ {
		switch n.FieldNameForChild(i) {
		case "name", "condition", "alternative", "value", "path":
			continue
		}
		c := n.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		if err := p.walk(path, src, c); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) walk(path, src string, n treesitter.Node) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.IsNull() || !n.IsNamed() {
		return nil
	}
	if strings.HasPrefix(n.Type(), "#") {
		return nil
	}
	switch n.Type() {
	case "comment", "preproc_call":
		return nil
	case "translation_unit", "preproc_else", "linkage_specification":
		return p.walkBody(path, src, n)
	case "preproc_ifdef":
		return p.walkIfdef(path, src, n)
	case "preproc_if", "preproc_elif":
		return p.walkIf(path, src, n)
	case "preproc_def":
		return p.walkDef(src, n)
	case "preproc_function_def":
		return p.walkFnMacro(src, n)
	case "preproc_include":
		return p.walkInclude(path, src, n)
	case "type_definition":
		return p.walkTypedef(path, src, n)
	case "declaration", "function_definition":
		return p.walkDecl(path, src, n)
	case "struct_specifier":
		_, err := p.aggregate(src, n, kindStruct)
		return err
	case "union_specifier":
		_, err := p.aggregate(src, n, kindUnion)
		return err
	case "enum_specifier":
		_, err := p.enumType(src, n)
		return err
	default:
		if n.IsError() || n.Type() == "ERROR" {
			return p.fail(path, src, n, errSyntax)
		}
		return p.fail(path, src, n, fmt.Errorf("%w: %s", errSyntax, n.Type()))
	}
}

func (p *parser) fail(path, src string, n treesitter.Node, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s:%d: %w", path, lineAt(src, n), err)
}

func (p *parser) walkIfdef(path, src string, n treesitter.Node) error {
	name := nodeText(src, childField(n, "name"))
	kind := ""
	if n.ChildCount() > 0 {
		kind = nodeText(src, n.Child(0))
	}
	on := p.defined(name)
	if strings.Contains(kind, "ifndef") {
		on = !on
	}
	if on {
		return p.walkBody(path, src, n)
	}
	if alt := childField(n, "alternative"); alt != nil && !alt.IsNull() {
		return p.walk(path, src, alt)
	}
	return nil
}

func (p *parser) walkIf(path, src string, n treesitter.Node) error {
	p.inIf = true
	v, err := p.eval(src, childField(n, "condition"))
	p.inIf = false
	if err != nil {
		return p.fail(path, src, n, err)
	}
	if v != 0 {
		return p.walkBody(path, src, n)
	}
	if alt := childField(n, "alternative"); alt != nil && !alt.IsNull() {
		return p.walk(path, src, alt)
	}
	return nil
}

func (p *parser) walkDef(src string, n treesitter.Node) error {
	name := nodeText(src, childField(n, "name"))
	if name == "" {
		return nil
	}
	val := ""
	if v := childField(n, "value"); v != nil && !v.IsNull() {
		val = nodeText(src, v)
	}
	p.define(name, val, true)
	return nil
}

func (p *parser) walkFnMacro(src string, n treesitter.Node) error {
	name := nodeText(src, childField(n, "name"))
	if name == "" {
		for i := uint32(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c != nil && !c.IsNull() && c.Type() == "identifier" {
				name = nodeText(src, c)
				break
			}
		}
	}
	if name != "" {
		p.macros[name] = &macro{fn: true, header: true, text: "0"}
	}
	return nil
}

func (p *parser) walkInclude(path, src string, n treesitter.Node) error {
	pn := childField(n, "path")
	if pn == nil || pn.IsNull() {
		return p.fail(path, src, n, errInclude)
	}
	switch pn.Type() {
	case "string_literal":
		rel := includePath(src, pn)
		found, err := p.search(path, rel, true)
		if err != nil {
			return err
		}
		if found == "" {
			return p.fail(path, src, n, fmt.Errorf("%w: %s", errInclude, rel))
		}
		return p.parseFile(found)
	case "system_lib_string":
		rel := strings.TrimSpace(nodeText(src, pn))
		rel = strings.TrimPrefix(rel, "<")
		rel = strings.TrimSuffix(rel, ">")
		found, err := p.search(path, rel, false)
		if err != nil {
			return err
		}
		if found == "" {
			return nil
		}
		return p.parseFile(found)
	default:
		return p.fail(path, src, n, fmt.Errorf("%w: %s", errInclude, pn.Type()))
	}
}

func includePath(src string, n treesitter.Node) string {
	if c := findNode(n, "string_content"); c != nil {
		return nodeText(src, c)
	}
	s := strings.TrimSpace(nodeText(src, n))
	return strings.Trim(s, `"`)
}

func (p *parser) search(from, rel string, quoted bool) (string, error) {
	if rel == "" {
		return "", errInclude
	}
	var dirs []string
	if quoted {
		dirs = append(dirs, filepath.Dir(from))
	}
	dirs = append(dirs, p.include...)
	for _, dir := range dirs {
		cand := filepath.Join(dir, rel)
		st, err := os.Stat(cand)
		if err == nil && !st.IsDir() {
			return cand, nil
		}
	}
	return "", nil
}

func (p *parser) note(t *Type) {
	if t == nil || t.listed {
		return
	}
	t.listed = true
	p.unit.Types = append(p.unit.Types, t)
}

func (p *parser) pointer(elem *Type) *Type {
	return &Type{
		Kind: kindPtr, Elem: elem,
		Size: p.target.Ptr, Align: p.target.Ptr,
		laid: true, natKnown: true, nat: true,
	}
}

func (p *parser) decay(t *Type) *Type {
	if t == nil {
		return nil
	}
	switch t.Kind {
	case kindArray:
		return p.pointer(t.Elem)
	case kindFunc:
		return p.pointer(t)
	default:
		return t
	}
}

func (p *parser) typeOf(src string, n treesitter.Node) (*Type, error) {
	if n == nil || n.IsNull() {
		return nil, errType
	}
	switch n.Type() {
	case "primitive_type":
		return p.lookupPrimitive(nodeText(src, n))
	case "sized_type_specifier":
		return p.sized(src, n)
	case "struct_specifier":
		return p.aggregate(src, n, kindStruct)
	case "union_specifier":
		return p.aggregate(src, n, kindUnion)
	case "enum_specifier":
		return p.enumType(src, n)
	case "type_identifier":
		name := nodeText(src, n)
		if t, ok := p.typedefs[name]; ok {
			return t, nil
		}
		return nil, fmt.Errorf("%w: %s", errType, name)
	default:
		return nil, fmt.Errorf("%w: %s", errType, n.Type())
	}
}

func (p *parser) lookupPrimitive(name string) (*Type, error) {
	switch name {
	case "void":
		return p.tyVoid, nil
	case "bool", "_Bool":
		return p.tyBool, nil
	case "char":
		return p.tyChar, nil
	case "short":
		return p.tyShort, nil
	case "int":
		return p.tyInt, nil
	case "long":
		return p.tyLong, nil
	case "float":
		return p.tyFloat, nil
	case "double":
		return p.tyDouble, nil
	case "signed":
		return p.tyInt, nil
	case "unsigned":
		return p.tyUInt, nil
	default:
		if t, ok := p.typedefs[name]; ok {
			return t, nil
		}
		return nil, fmt.Errorf("%w: %s", errType, name)
	}
}

func (p *parser) sized(src string, n treesitter.Node) (*Type, error) {
	longs, shorts := 0, 0
	unsigned := false
	base := ""
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() {
			continue
		}
		switch c.Type() {
		case "long":
			longs++
		case "short":
			shorts++
		case "unsigned":
			unsigned = true
		case "signed":
		case "primitive_type":
			base = nodeText(src, c)
		}
	}
	switch base {
	case "double":
		if longs > 0 {
			return p.tyLongDouble, nil
		}
		return p.tyDouble, nil
	case "float":
		return p.tyFloat, nil
	case "char":
		if unsigned {
			return p.tyUChar, nil
		}
		return p.tySChar, nil
	}
	switch {
	case longs >= 2 && unsigned:
		return p.tyULongLong, nil
	case longs >= 2:
		return p.tyLongLong, nil
	case longs == 1 && unsigned:
		return p.tyULong, nil
	case longs == 1:
		return p.tyLong, nil
	case shorts > 0 && unsigned:
		return p.tyUShort, nil
	case shorts > 0:
		return p.tyShort, nil
	case unsigned:
		return p.tyUInt, nil
	default:
		return p.tyInt, nil
	}
}

func (p *parser) aggregate(src string, n treesitter.Node, k kind) (*Type, error) {
	word := "struct"
	if k == kindUnion {
		word = "union"
	}
	tag := ""
	if name := childField(n, "name"); name != nil && !name.IsNull() {
		tag = nodeText(src, name)
	}
	var typ *Type
	if tag != "" {
		key := word + " " + tag
		typ = p.tags[key]
		if typ == nil {
			typ = &Type{Kind: k, Tag: tag, Spell: tag, prefer: tag, Incomplete: true}
			p.tags[key] = typ
			p.note(typ)
		}
	} else {
		typ = &Type{Kind: k, Spell: word, Incomplete: true}
		p.note(typ)
	}
	packed, align := specAttrs(src, n)
	if packed {
		typ.Packed = true
	}
	if align > typ.AlignAttr {
		typ.AlignAttr = align
	}
	body := childField(n, "body")
	if body == nil || body.IsNull() {
		return typ, nil
	}
	if !typ.Incomplete {
		return typ, nil
	}
	typ.Incomplete = false
	fields, err := p.fieldList(src, typ, body)
	if err != nil {
		return nil, err
	}
	typ.Fields = fields
	return typ, nil
}

func (p *parser) fieldList(src string, parent *Type, body treesitter.Node) ([]*Field, error) {
	var out []*Field
	for i := uint32(0); i < body.ChildCount(); i++ {
		c := body.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		if c.Type() == "comment" {
			continue
		}
		if c.Type() != "field_declaration" {
			return nil, fmt.Errorf("%w: %s", errSyntax, c.Type())
		}
		fields, err := p.fieldDecl(src, parent, c)
		if err != nil {
			return nil, err
		}
		out = append(out, fields...)
	}
	return out, nil
}

func (p *parser) fieldDecl(src string, parent *Type, n treesitter.Node) ([]*Field, error) {
	base, err := p.typeOf(src, childField(n, "type"))
	if err != nil {
		return nil, err
	}
	packed, align := specAttrs(src, n)
	width, isBit, err := p.bitWidth(src, n)
	if err != nil {
		return nil, err
	}
	decls := childFields(n, "declarator")
	if len(decls) == 0 {
		f := &Field{Type: base, Packed: packed, AlignAttr: align, IsBit: isBit, BitWidth: width}
		return []*Field{f}, nil
	}
	var out []*Field
	for _, d := range decls {
		typ, name, err := p.applyDecl(src, base, d)
		if err != nil {
			return nil, err
		}
		if name != "" && typ != nil && typ.Tag == "" && (typ.Kind == kindStruct || typ.Kind == kindUnion) {
			typ.prefer = typeLabel(parent) + "_" + name
		}
		out = append(out, &Field{
			Name: name, Type: typ, Packed: packed, AlignAttr: align,
			IsBit: isBit, BitWidth: width,
		})
	}
	return out, nil
}

func typeLabel(t *Type) string {
	if t == nil {
		return "Anon"
	}
	if t.prefer != "" {
		return t.prefer
	}
	if t.Tag != "" {
		return t.Tag
	}
	return "Anon"
}

func (p *parser) bitWidth(src string, n treesitter.Node) (int, bool, error) {
	var clause treesitter.Node
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c != nil && !c.IsNull() && c.Type() == "bitfield_clause" {
			clause = c
			break
		}
	}
	if clause == nil {
		return 0, false, nil
	}
	for i := uint32(0); i < clause.ChildCount(); i++ {
		c := clause.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		v, err := p.eval(src, c)
		if err != nil {
			return 0, true, err
		}
		return int(v), true, nil
	}
	return 0, true, fmt.Errorf("%w: bit-field", errExpr)
}

func (p *parser) enumType(src string, n treesitter.Node) (*Type, error) {
	tag := ""
	if name := childField(n, "name"); name != nil && !name.IsNull() {
		tag = nodeText(src, name)
	}
	var typ *Type
	if tag != "" {
		key := "enum " + tag
		typ = p.tags[key]
		if typ == nil {
			typ = &Type{
				Kind: kindEnum, Tag: tag, Spell: tag, prefer: tag,
				Size: p.target.Int, Align: p.target.Int, laid: true,
			}
			p.tags[key] = typ
			p.note(typ)
		}
	} else {
		typ = &Type{Kind: kindEnum, Spell: "enum", Size: p.target.Int, Align: p.target.Int, laid: true}
		p.note(typ)
	}
	body := childField(n, "body")
	if body == nil || body.IsNull() || len(typ.Enums) > 0 {
		return typ, nil
	}
	next := int64(0)
	for i := uint32(0); i < body.ChildCount(); i++ {
		c := body.Child(i)
		if c == nil || c.IsNull() || c.Type() != "enumerator" {
			continue
		}
		name := nodeText(src, childField(c, "name"))
		v := next
		if val := childField(c, "value"); val != nil && !val.IsNull() {
			got, err := p.eval(src, val)
			if err != nil {
				return nil, err
			}
			v = got
		}
		next = v + 1
		typ.Enums = append(typ.Enums, Enumerator{Name: name, Value: v})
		p.enums[name] = v
	}
	return typ, nil
}

func (p *parser) applyDecl(src string, base *Type, n treesitter.Node) (*Type, string, error) {
	if n == nil || n.IsNull() {
		return base, "", nil
	}
	switch n.Type() {
	case "identifier", "field_identifier", "type_identifier", "primitive_type":
		return base, nodeText(src, n), nil
	case "init_declarator":
		return p.applyDecl(src, base, childField(n, "declarator"))
	case "pointer_declarator", "abstract_pointer_declarator":
		inner := innerDecl(n)
		ptr := p.pointer(base)
		if inner == nil {
			return ptr, "", nil
		}
		return p.applyDecl(src, ptr, inner)
	case "array_declarator", "abstract_array_declarator":
		ln := -1
		if sz := childField(n, "size"); sz != nil && !sz.IsNull() {
			v, err := p.eval(src, sz)
			if err != nil {
				return nil, "", err
			}
			if v < 0 {
				return nil, "", fmt.Errorf("%w: array length", errExpr)
			}
			ln = int(v)
			if ln == 0 {
				ln = -1
			}
		}
		arr := &Type{Kind: kindArray, Elem: base, Len: ln}
		inner := childField(n, "declarator")
		if inner == nil || inner.IsNull() {
			return arr, "", nil
		}
		return p.applyDecl(src, arr, inner)
	case "function_declarator", "abstract_function_declarator":
		params, variadic, err := p.params(src, childField(n, "parameters"))
		if err != nil {
			return nil, "", err
		}
		fn := &Type{
			Kind: kindFunc, Elem: base, Params: params, Variadic: variadic,
			Size: p.target.Ptr, Align: p.target.Ptr, laid: true, natKnown: true, nat: true,
		}
		inner := childField(n, "declarator")
		if inner == nil || inner.IsNull() {
			return fn, "", nil
		}
		return p.applyDecl(src, fn, inner)
	case "parenthesized_declarator", "abstract_parenthesized_declarator":
		return p.applyDecl(src, base, innerDecl(n))
	default:
		return nil, "", fmt.Errorf("%w: declarator %s", errType, n.Type())
	}
}

func (p *parser) params(src string, list treesitter.Node) ([]Param, bool, error) {
	if list == nil || list.IsNull() {
		return nil, false, nil
	}
	variadic := strings.Contains(nodeText(src, list), "...")
	var out []Param
	for i := uint32(0); i < list.ChildCount(); i++ {
		c := list.Child(i)
		if c == nil || c.IsNull() || c.Type() != "parameter_declaration" {
			continue
		}
		base, err := p.typeOf(src, childField(c, "type"))
		if err != nil {
			return nil, false, err
		}
		decl := childField(c, "declarator")
		if (decl == nil || decl.IsNull()) && base.Kind == kindVoid {
			continue
		}
		typ := base
		name := ""
		if decl != nil && !decl.IsNull() {
			typ, name, err = p.applyDecl(src, base, decl)
			if err != nil {
				return nil, false, err
			}
		}
		out = append(out, Param{Name: name, Type: p.decay(typ)})
	}
	return out, variadic, nil
}

func (p *parser) walkTypedef(path, src string, n treesitter.Node) error {
	base, err := p.typeOf(src, childField(n, "type"))
	if err != nil {
		return p.fail(path, src, n, err)
	}
	typ, name, err := p.applyDecl(src, base, childField(n, "declarator"))
	if err != nil {
		return p.fail(path, src, n, err)
	}
	if name == "" {
		return p.fail(path, src, n, errType)
	}
	if _, exists := p.typedefs[name]; exists {
		return nil
	}
	if typ == base && (base.Kind == kindStruct || base.Kind == kindUnion || base.Kind == kindEnum) {
		base.prefer = name
		base.Spell = name
		p.typedefs[name] = base
		return nil
	}
	alias := &Type{Kind: kindAlias, Elem: typ, Spell: name, prefer: name}
	p.typedefs[name] = alias
	p.note(alias)
	return nil
}

func (p *parser) walkDecl(path, src string, n treesitter.Node) error {
	base, err := p.typeOf(src, childField(n, "type"))
	if err != nil {
		return p.fail(path, src, n, err)
	}
	decls := childFields(n, "declarator")
	if len(decls) == 0 {
		return nil
	}
	skip := hasStatic(src, n)
	for _, d := range decls {
		typ, name, err := p.applyDecl(src, base, d)
		if err != nil {
			return p.fail(path, src, n, err)
		}
		if name == "" || typ == nil {
			continue
		}
		if typ.Kind == kindFunc {
			if skip {
				continue
			}
			if typ.Variadic {
				p.unit.Skipped = append(p.unit.Skipped, name)
				continue
			}
			p.unit.Funcs = append(p.unit.Funcs, Func{Name: name, Ret: typ.Elem, Params: typ.Params})
			continue
		}
		if skip {
			continue
		}
		p.unit.Objects = append(p.unit.Objects, Object{Name: name, Type: typ})
	}
	return nil
}

func hasStatic(src string, n treesitter.Node) bool {
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() || c.Type() != "storage_class_specifier" {
			continue
		}
		if strings.Contains(nodeText(src, c), "static") {
			return true
		}
	}
	return false
}

func specAttrs(src string, n treesitter.Node) (packed bool, align int) {
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() || c.Type() != "attribute_specifier" {
			continue
		}
		p, a := readAttr(src, c)
		packed = packed || p
		if a > align {
			align = a
		}
	}
	return packed, align
}

func readAttr(src string, n treesitter.Node) (packed bool, align int) {
	if n == nil || n.IsNull() {
		return false, 0
	}
	if n.Type() == "identifier" && nodeText(src, n) == "packed" {
		packed = true
	}
	if n.Type() == "call_expression" {
		fn := childField(n, "function")
		if fn != nil && nodeText(src, fn) == "aligned" {
			if v, err := attrNumber(src, childField(n, "arguments")); err == nil && v > 0 {
				align = int(v)
			}
		}
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		p, a := readAttr(src, n.Child(i))
		packed = packed || p
		if a > align {
			align = a
		}
	}
	return packed, align
}

func attrNumber(src string, n treesitter.Node) (int64, error) {
	if n == nil || n.IsNull() {
		return 0, errExpr
	}
	if n.Type() == "number_literal" {
		return parseNumber(nodeText(src, n))
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		if v, err := attrNumber(src, n.Child(i)); err == nil {
			return v, nil
		}
	}
	return 0, errExpr
}

func (p *parser) layoutOne(t *Type) error {
	if t == nil {
		return nil
	}
	if (t.Kind == kindStruct || t.Kind == kindUnion) && t.Incomplete {
		t.Size = 0
		t.Align = 1
		t.laid = true
		return nil
	}
	return p.target.layout(t)
}

func (p *parser) finish() error {
	for _, m := range p.macros {
		if !m.header || m.fn || strings.TrimSpace(m.text) == "" || m.known {
			continue
		}
		if v, err := p.evalSnippet(m.text); err == nil {
			m.known = true
			m.value = v
		}
	}
	names := make([]string, 0, len(p.macros))
	for name, m := range p.macros {
		if m.header && !m.fn && m.known {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		p.unit.Consts = append(p.unit.Consts, Const{Name: name, Value: p.macros[name].value})
	}
	limit := int64(1) << (p.target.Int*8 - 1)
	for _, t := range p.unit.Types {
		for _, e := range t.Enums {
			if e.Value >= limit || e.Value < -limit {
				return fmt.Errorf("%w: enum %s", errExpr, e.Name)
			}
		}
		if err := p.layoutOne(t); err != nil {
			return err
		}
	}
	for i := range p.unit.Funcs {
		fn := &p.unit.Funcs[i]
		if err := p.layoutOne(fn.Ret); err != nil {
			return err
		}
		for _, param := range fn.Params {
			if err := p.layoutOne(param.Type); err != nil {
				return err
			}
		}
	}
	for _, obj := range p.unit.Objects {
		if err := p.layoutOne(obj.Type); err != nil {
			return err
		}
	}
	return nil
}
