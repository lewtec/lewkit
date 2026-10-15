package c

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func parseSource(src []byte) (*File, error) {
	p := &parser{
		src:      splice(src),
		line:     1,
		bol:      true,
		file:     New(),
		tags:     map[string]*Type{},
		typedefs: map[string]*Type{},
		consts:   map[string]int64{},
	}
	p.next()
	for p.tok.kind != tokEOF {
		switch p.tok.kind {
		case tokInclude:
			p.file.Decls = append(p.file.Decls, &Decl{
				Kind:    DeclInclude,
				Include: &Include{Path: p.tok.text, System: p.tok.system},
			})
			p.next()
		case tokDefine:
			if err := p.define(p.tok.text, p.tok.alt); err != nil {
				return nil, err
			}
			p.next()
		default:
			if p.tok.kind == tokPunct && strings.HasPrefix(p.tok.text, "#") {
				return nil, p.unsupported(p.tok.text)
			}
			if err := p.decl(); err != nil {
				return nil, err
			}
		}
	}
	return p.file, nil
}

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

type tokKind uint8

const (
	tokEOF tokKind = iota
	tokIdent
	tokNumber
	tokString
	tokChar
	tokPunct
	tokInclude
	tokDefine
)

type token struct {
	kind     tokKind
	text     string
	alt      string
	system   bool
	line     int
	intVal   int64
	floatVal float64
	isFloat  bool
}

type parser struct {
	src      []byte
	i        int
	line     int
	bol      bool
	tok      token
	file     *File
	tags     map[string]*Type
	typedefs map[string]*Type
	consts   map[string]int64
}

type parsedSpec struct {
	typ     *Type
	raw     *Type
	storage Storage
	typedef bool
	inline  bool
	body    bool
}

func (p *parser) decl() error {
	spec, err := p.specifiers()
	if err != nil {
		return err
	}
	if spec.typ == nil {
		return p.bad("declaration")
	}
	if spec.body && !spec.typedef && spec.raw != nil && spec.raw.home == nil && spec.raw.Tag != "" {
		p.defineTag(spec.raw)
	}
	if p.punct(";") {
		if spec.typedef {
			return p.bad("typedef")
		}
		if !isAggregate(spec.raw) {
			return p.bad("declaration")
		}
		if spec.raw.home == nil {
			define := spec.body
			p.file.Decls = append(p.file.Decls, &Decl{
				Kind: DeclType,
				Type: &TypeDecl{Type: spec.raw, Define: define},
			})
			if define {
				spec.raw.home = p.file.Decls[len(p.file.Decls)-1].Type
			}
		}
		return nil
	}
	if spec.typedef {
		return p.typedefDecl(spec)
	}
	typ, name, err := p.declarator(spec.typ)
	if err != nil {
		return err
	}
	if typ != nil && typ.Kind == KindFunction && (p.at("{") || p.at(";")) {
		return p.function(spec, typ, name)
	}
	return p.objects(spec, typ, name)
}

func (p *parser) typedefDecl(spec parsedSpec) error {
	typ, name, err := p.declarator(spec.typ)
	if err != nil {
		return err
	}
	if name == "" {
		return p.bad("typedef name")
	}
	if !p.punct(";") {
		return p.bad(";")
	}
	define := spec.body && sameAggregate(typ, spec.raw)
	decl := &TypeDecl{Name: name, Type: typ, Define: define}
	if define && spec.raw != nil {
		spec.raw.home = decl
	}
	p.file.Decls = append(p.file.Decls, &Decl{Kind: DeclType, Type: decl})
	p.typedefs[name] = useType(name, typ)
	return nil
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

func (p *parser) function(spec parsedSpec, typ *Type, name string) error {
	fn := &Func{
		Name:     name,
		Result:   typ.Elem,
		Params:   typ.Params,
		Variadic: typ.Variadic,
		Storage:  spec.storage,
		Inline:   spec.inline,
	}
	if p.punct(";") {
		p.file.Decls = append(p.file.Decls, &Decl{Kind: DeclFunc, Func: fn})
		return nil
	}
	body, err := p.statement()
	if err != nil {
		return err
	}
	fn.Body = body
	p.file.Decls = append(p.file.Decls, &Decl{Kind: DeclFunc, Func: fn})
	return nil
}

func (p *parser) objects(spec parsedSpec, typ *Type, name string) error {
	for {
		obj := &Object{Name: name, Type: typ, Storage: spec.storage}
		if p.punct("=") {
			init, err := p.expr()
			if err != nil {
				return err
			}
			obj.Init = init
		}
		p.file.Decls = append(p.file.Decls, &Decl{Kind: DeclObject, Object: obj})
		if !p.punct(",") {
			break
		}
		var err error
		typ, name, err = p.declarator(spec.typ)
		if err != nil {
			return err
		}
	}
	if !p.punct(";") {
		return p.bad(";")
	}
	return nil
}

func (p *parser) define(name, value string) error {
	if name == "" {
		return p.bad("define")
	}
	if strings.Contains(name, "(") {
		return p.unsupported("function macro")
	}
	val := int64(0)
	if strings.TrimSpace(value) != "" {
		got, err := p.evalText(value)
		if err != nil {
			return err
		}
		val = got
	}
	p.consts[name] = val
	p.file.Decls = append(p.file.Decls, &Decl{
		Kind:  DeclConst,
		Const: &Const{Name: name, Value: val},
	})
	return nil
}

func (p *parser) evalText(text string) (int64, error) {
	sub := &parser{
		src:      []byte(text),
		line:     p.tok.line,
		bol:      true,
		tags:     p.tags,
		typedefs: p.typedefs,
		consts:   p.consts,
	}
	sub.next()
	e, err := sub.expr()
	if err != nil {
		return 0, err
	}
	if sub.tok.kind != tokEOF {
		return 0, sub.bad("constant")
	}
	v, ok := evalExpr(e, p.consts)
	if !ok {
		return 0, p.bad("constant")
	}
	return v, nil
}

func (p *parser) defineTag(t *Type) {
	if t == nil || t.Tag == "" {
		return
	}
	if t.home != nil && t.home.Define {
		return
	}
	decl := &TypeDecl{Type: t, Define: true}
	t.home = decl
	t.Inline = false
	p.file.Decls = append(p.file.Decls, &Decl{Kind: DeclType, Type: decl})
}

func (p *parser) specifiers() (parsedSpec, error) {
	var (
		sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble bool
		sawSigned, sawUnsigned                                 bool
		longs, shorts                                          int
		isConst, isVolatile                                    bool
		storage                                                Storage
		typedef, inline                                        bool
		aggr                                                   *Type
		named                                                  *Type
		body                                                   bool
		saw                                                    bool
	)
loop:
	for p.tok.kind == tokIdent {
		word := p.tok.text
		switch word {
		case "const":
			isConst = true
			p.next()
		case "volatile":
			isVolatile = true
			p.next()
		case "static":
			if storage != 0 || typedef {
				return parsedSpec{}, p.bad("storage class")
			}
			storage = StorageStatic
			p.next()
		case "extern":
			if storage != 0 || typedef {
				return parsedSpec{}, p.bad("storage class")
			}
			storage = StorageExtern
			p.next()
		case "typedef":
			if storage != 0 || typedef {
				return parsedSpec{}, p.bad("storage class")
			}
			typedef = true
			p.next()
		case "inline":
			inline = true
			p.next()
		case "void":
			sawVoid = true
			saw = true
			p.next()
		case "bool", "_Bool":
			sawBool = true
			saw = true
			p.next()
		case "char":
			sawChar = true
			saw = true
			p.next()
		case "short":
			shorts++
			saw = true
			p.next()
		case "int":
			sawInt = true
			saw = true
			p.next()
		case "long":
			longs++
			saw = true
			p.next()
		case "float":
			sawFloat = true
			saw = true
			p.next()
		case "double":
			sawDouble = true
			saw = true
			p.next()
		case "signed":
			sawSigned = true
			saw = true
			p.next()
		case "unsigned":
			sawUnsigned = true
			saw = true
			p.next()
		case "struct", "union", "enum":
			if saw || aggr != nil || named != nil {
				return parsedSpec{}, p.bad(word)
			}
			p.next()
			t, defined, err := p.aggregate(word, typedef)
			if err != nil {
				return parsedSpec{}, err
			}
			aggr = t
			body = defined
			saw = true
		case "restrict", "auto", "register", "_Noreturn", "_Thread_local", "_Alignas", "_Atomic", "_Generic":
			return parsedSpec{}, p.unsupported(word)
		default:
			if !saw && aggr == nil && named == nil {
				if t, ok := p.typedefs[word]; ok {
					named = t
					saw = true
					p.next()
					continue
				}
			}
			break loop
		}
	}
	spec := parsedSpec{storage: storage, typedef: typedef, inline: inline, body: body}
	switch {
	case aggr != nil:
		spec.raw = aggr
		spec.typ = qualify(aggr, isConst, isVolatile)
	case named != nil:
		n := *named
		n.Const = n.Const || isConst
		n.Volatile = n.Volatile || isVolatile
		n.home = nil
		n.laying = false
		spec.typ = &n
		spec.raw = named
	default:
		t, err := p.scalar(sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned, longs, shorts)
		if err != nil {
			return parsedSpec{}, err
		}
		t.Const = isConst
		t.Volatile = isVolatile
		spec.typ = t
		spec.raw = t
	}
	return spec, nil
}

func (p *parser) scalar(sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned bool, longs, shorts int) (*Type, error) {
	switch {
	case sawVoid:
		return Void(), nil
	case sawBool:
		return Bool(), nil
	case sawFloat && longs > 0:
		return nil, p.bad("float")
	case sawFloat:
		return Float(), nil
	case sawDouble && longs > 0:
		return LongDouble(), nil
	case sawDouble:
		return Double(), nil
	}
	if !sawChar && !sawInt && !sawSigned && !sawUnsigned && longs == 0 && shorts == 0 {
		return nil, p.bad("type")
	}
	unsigned := sawUnsigned && !sawSigned
	switch {
	case sawChar && longs == 0 && shorts == 0:
		if sawUnsigned {
			return UnsignedChar(), nil
		}
		if sawSigned {
			return SignedChar(), nil
		}
		return Char(), nil
	case longs >= 2:
		if unsigned {
			return UnsignedLongLong(), nil
		}
		return LongLong(), nil
	case longs == 1:
		if unsigned {
			return UnsignedLong(), nil
		}
		return Long(), nil
	case shorts > 0:
		if unsigned {
			return UnsignedShort(), nil
		}
		return Short(), nil
	case unsigned:
		return UnsignedInt(), nil
	default:
		return Int(), nil
	}
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

func (p *parser) aggregate(word string, typedef bool) (*Type, bool, error) {
	kind := KindStruct
	switch word {
	case "union":
		kind = KindUnion
	case "enum":
		kind = KindEnum
	}
	tag := ""
	if p.tok.kind == tokIdent && !isKeyword(p.tok.text) {
		tag = p.tok.text
		p.next()
	}
	if !p.at("{") {
		if tag == "" {
			return nil, false, p.bad(word)
		}
		return p.tag(kind, tag), false, nil
	}
	if !p.punct("{") {
		return nil, false, p.bad("{")
	}
	var t *Type
	if tag == "" {
		t = &Type{Kind: kind, Inline: true}
	} else {
		t = p.tag(kind, tag)
		if t.Defined {
			return nil, false, p.bad("redefinition")
		}
	}
	if kind == KindEnum {
		if err := p.enumerators(t); err != nil {
			return nil, false, err
		}
	} else if err := p.fields(t); err != nil {
		return nil, false, err
	}
	if !p.punct("}") {
		return nil, false, p.bad("}")
	}
	t.Defined = true
	if tag == "" {
		t.Inline = true
	}
	layout(t)
	if tag != "" && !typedef {
		p.defineTag(t)
	}
	return t, true, nil
}

func (p *parser) tag(kind Kind, tag string) *Type {
	key := aggregateWord(kind) + " " + tag
	if t := p.tags[key]; t != nil {
		return t
	}
	t := &Type{Kind: kind, Tag: tag}
	if kind == KindEnum {
		t.Size = intSize
		t.Align = intSize
	}
	p.tags[key] = t
	return t
}

func (p *parser) fields(parent *Type) error {
	for !p.at("}") && p.tok.kind != tokEOF {
		spec, err := p.specifiers()
		if err != nil {
			return err
		}
		if p.at(";") {
			p.next()
			parent.Fields = append(parent.Fields, &Field{Type: spec.typ})
			continue
		}
		for {
			if p.at(":") {
				return p.unsupported("bit-field")
			}
			typ, name, err := p.declarator(spec.typ)
			if err != nil {
				return err
			}
			if p.at(":") {
				return p.unsupported("bit-field")
			}
			parent.Fields = append(parent.Fields, &Field{Name: name, Type: typ})
			if !p.punct(",") {
				break
			}
		}
		if !p.punct(";") {
			return p.bad(";")
		}
	}
	return nil
}

func (p *parser) enumerators(t *Type) error {
	next := int64(0)
	if p.at("}") {
		return nil
	}
	for {
		name, ok := p.ident()
		if !ok {
			return p.bad("enumerator")
		}
		val := next
		if p.punct("=") {
			e, err := p.expr()
			if err != nil {
				return err
			}
			got, ok := evalExpr(e, p.consts)
			if !ok {
				return p.bad("enumerator")
			}
			val = got
		}
		next = val + 1
		t.Enums = append(t.Enums, &Enumerator{Name: name, Value: val})
		p.consts[name] = val
		if p.punct(",") {
			if p.at("}") {
				break
			}
			continue
		}
		break
	}
	return nil
}

type declBuild func(*Type) *Type

func (p *parser) declarator(base *Type) (*Type, string, error) {
	name, build, err := p.parseDeclarator()
	if err != nil {
		return nil, "", err
	}
	return build(base), name, nil
}

func (p *parser) parseDeclarator() (string, declBuild, error) {
	ptrs, err := p.pointers()
	if err != nil {
		return "", nil, err
	}
	name, direct, err := p.direct()
	if err != nil {
		return "", nil, err
	}
	return name, func(base *Type) *Type {
		return direct(ptrs(base))
	}, nil
}

func (p *parser) pointers() (declBuild, error) {
	type qual struct{ c, v bool }
	var qs []qual
	for p.punct("*") {
		c, v := false, false
		for p.tok.kind == tokIdent {
			switch p.tok.text {
			case "const":
				c = true
				p.next()
			case "volatile":
				v = true
				p.next()
			case "restrict":
				return nil, p.unsupported("restrict")
			default:
				goto done
			}
		}
	done:
		qs = append(qs, qual{c, v})
	}
	return func(base *Type) *Type {
		t := base
		for _, q := range qs {
			t = &Type{Kind: KindPointer, Elem: t, Const: q.c, Volatile: q.v, Size: ptrSize, Align: ptrSize}
		}
		return t
	}, nil
}

func (p *parser) direct() (string, declBuild, error) {
	var name string
	var inner declBuild
	switch {
	case p.punct("("):
		var err error
		name, inner, err = p.parseDeclarator()
		if err != nil {
			return "", nil, err
		}
		if !p.punct(")") {
			return "", nil, p.bad(")")
		}
	case p.tok.kind == tokIdent && !isKeyword(p.tok.text):
		name = p.tok.text
		p.next()
		inner = func(t *Type) *Type { return t }
	default:
		inner = func(t *Type) *Type { return t }
	}
	var suffixes []declBuild
	for {
		switch {
		case p.punct("["):
			n := -1
			if !p.at("]") {
				e, err := p.expr()
				if err != nil {
					return "", nil, err
				}
				v, ok := evalExpr(e, p.consts)
				if !ok || v < 0 {
					return "", nil, p.bad("array length")
				}
				n = int(v)
			}
			if !p.punct("]") {
				return "", nil, p.bad("]")
			}
			length := n
			suffixes = append(suffixes, func(t *Type) *Type {
				a := &Type{Kind: KindArray, Elem: t, Len: length}
				layout(a)
				return a
			})
		case p.punct("("):
			params, variadic, err := p.parameters()
			if err != nil {
				return "", nil, err
			}
			suffixes = append(suffixes, func(t *Type) *Type {
				return &Type{Kind: KindFunction, Elem: t, Params: params, Variadic: variadic, Align: 1}
			})
		default:
			return name, func(base *Type) *Type {
				t := base
				for i := len(suffixes) - 1; i >= 0; i-- {
					t = suffixes[i](t)
				}
				return inner(t)
			}, nil
		}
	}
}

func (p *parser) parameters() ([]*Param, bool, error) {
	if p.punct(")") {
		return nil, false, nil
	}
	var params []*Param
	for {
		if p.punct("...") {
			if !p.punct(")") {
				return nil, false, p.bad(")")
			}
			return params, true, nil
		}
		spec, err := p.specifiers()
		if err != nil {
			return nil, false, err
		}
		typ := spec.typ
		name := ""
		if p.startsDeclarator() {
			typ, name, err = p.declarator(spec.typ)
			if err != nil {
				return nil, false, err
			}
		}
		if name == "" && typ != nil && typ.Kind == KindVoid && len(params) == 0 && p.at(")") {
			p.next()
			return nil, false, nil
		}
		params = append(params, &Param{Name: name, Type: typ})
		if p.punct(",") {
			continue
		}
		if !p.punct(")") {
			return nil, false, p.bad(")")
		}
		return params, false, nil
	}
}

func (p *parser) statement() (*Stmt, error) {
	switch {
	case p.at("{"):
		return p.block()
	case p.keyword("if"):
		return p.ifStmt()
	case p.keyword("while"):
		if !p.punct("(") {
			return nil, p.bad("(")
		}
		cond, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.punct(")") {
			return nil, p.bad(")")
		}
		body, err := p.statement()
		if err != nil {
			return nil, err
		}
		return While(cond, body), nil
	case p.keyword("for"):
		return p.forStmt()
	case p.keyword("return"):
		var e *Expr
		if !p.at(";") {
			var err error
			e, err = p.expr()
			if err != nil {
				return nil, err
			}
		}
		if !p.punct(";") {
			return nil, p.bad(";")
		}
		return Return(e), nil
	case p.keyword("break"):
		if !p.punct(";") {
			return nil, p.bad(";")
		}
		return Break(), nil
	case p.keyword("continue"):
		if !p.punct(";") {
			return nil, p.bad(";")
		}
		return Continue(), nil
	case p.atKeyword("switch"), p.atKeyword("do"), p.atKeyword("goto"):
		return nil, p.unsupported(p.tok.text)
	case p.startsDecl():
		return p.declStmt()
	default:
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.punct(";") {
			return nil, p.bad(";")
		}
		return ExprStmt(e), nil
	}
}

func (p *parser) block() (*Stmt, error) {
	if !p.punct("{") {
		return nil, p.bad("{")
	}
	var stmts []*Stmt
	for !p.at("}") && p.tok.kind != tokEOF {
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, s)
	}
	if !p.punct("}") {
		return nil, p.bad("}")
	}
	return Block(stmts...), nil
}

func (p *parser) ifStmt() (*Stmt, error) {
	if !p.punct("(") {
		return nil, p.bad("(")
	}
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	if !p.punct(")") {
		return nil, p.bad(")")
	}
	body, err := p.statement()
	if err != nil {
		return nil, err
	}
	s := If(cond, body)
	if p.keyword("else") {
		alt, err := p.statement()
		if err != nil {
			return nil, err
		}
		s.Else = alt
	}
	return s, nil
}

func (p *parser) forStmt() (*Stmt, error) {
	if !p.punct("(") {
		return nil, p.bad("(")
	}
	var init *Stmt
	switch {
	case p.punct(";"):
	case p.startsDecl():
		var err error
		init, err = p.declStmt()
		if err != nil {
			return nil, err
		}
	default:
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.punct(";") {
			return nil, p.bad(";")
		}
		init = ExprStmt(e)
	}
	var cond *Expr
	if !p.at(";") {
		var err error
		cond, err = p.expr()
		if err != nil {
			return nil, err
		}
	}
	if !p.punct(";") {
		return nil, p.bad(";")
	}
	var post *Expr
	if !p.at(")") {
		var err error
		post, err = p.expr()
		if err != nil {
			return nil, err
		}
	}
	if !p.punct(")") {
		return nil, p.bad(")")
	}
	body, err := p.statement()
	if err != nil {
		return nil, err
	}
	return For(init, cond, post, body), nil
}

func (p *parser) declStmt() (*Stmt, error) {
	spec, err := p.specifiers()
	if err != nil {
		return nil, err
	}
	if spec.typedef {
		return nil, p.unsupported("local typedef")
	}
	var objs []*Object
	for {
		typ, name, err := p.declarator(spec.typ)
		if err != nil {
			return nil, err
		}
		obj := &Object{Name: name, Type: typ, Storage: spec.storage}
		if p.punct("=") {
			init, err := p.expr()
			if err != nil {
				return nil, err
			}
			obj.Init = init
		}
		objs = append(objs, obj)
		if !p.punct(",") {
			break
		}
	}
	if !p.punct(";") {
		return nil, p.bad(";")
	}
	return &Stmt{Kind: StmtDecl, Objects: objs}, nil
}

func (p *parser) expr() (*Expr, error) {
	left, err := p.binary(0)
	if err != nil {
		return nil, err
	}
	if p.punct("=") {
		right, err := p.expr()
		if err != nil {
			return nil, err
		}
		return Assign(left, right), nil
	}
	if p.at("+=") || p.at("-=") || p.at("*=") || p.at("/=") || p.at("%=") || p.at("&=") || p.at("|=") || p.at("^=") || p.at("<<=") || p.at(">>=") {
		return nil, p.unsupported("assignment")
	}
	return left, nil
}

func (p *parser) binary(min int) (*Expr, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		op, prec, ok := binOp(p.tok)
		if !ok || prec < min {
			return left, nil
		}
		p.next()
		right, err := p.binary(prec + 1)
		if err != nil {
			return nil, err
		}
		left = Binary(op, left, right)
	}
}

func (p *parser) unary() (*Expr, error) {
	switch {
	case p.punct("&"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpAddr, x), nil
	case p.punct("*"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpDeref, x), nil
	case p.punct("!"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpNot, x), nil
	case p.punct("~"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpBitNot, x), nil
	case p.punct("-"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpNeg, x), nil
	case p.punct("+"):
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary(OpPos, x), nil
	case p.keyword("sizeof"):
		return p.sizeof()
	case p.at("++"), p.at("--"):
		return nil, p.unsupported(p.tok.text)
	default:
		return p.castOrPrimary()
	}
}

func (p *parser) sizeof() (*Expr, error) {
	if !p.punct("(") {
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return SizeofExpr(x), nil
	}
	if p.startsType() {
		spec, err := p.specifiers()
		if err != nil {
			return nil, err
		}
		typ := spec.typ
		if p.startsDeclarator() {
			var name string
			typ, name, err = p.declarator(spec.typ)
			if err != nil {
				return nil, err
			}
			if name != "" {
				return nil, p.bad("sizeof")
			}
		}
		if !p.punct(")") {
			return nil, p.bad(")")
		}
		return SizeofType(typ), nil
	}
	e, err := p.expr()
	if err != nil {
		return nil, err
	}
	if !p.punct(")") {
		return nil, p.bad(")")
	}
	return p.postfix(SizeofExpr(e))
}

func (p *parser) castOrPrimary() (*Expr, error) {
	if p.punct("(") {
		if p.startsType() {
			spec, err := p.specifiers()
			if err != nil {
				return nil, err
			}
			typ := spec.typ
			if p.startsDeclarator() {
				var name string
				typ, name, err = p.declarator(spec.typ)
				if err != nil {
					return nil, err
				}
				if name != "" {
					return nil, p.bad("cast")
				}
			}
			if !p.punct(")") {
				return nil, p.bad(")")
			}
			x, err := p.unary()
			if err != nil {
				return nil, err
			}
			return Cast(typ, x), nil
		}
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.punct(")") {
			return nil, p.bad(")")
		}
		return p.postfix(e)
	}
	e, err := p.primary()
	if err != nil {
		return nil, err
	}
	return p.postfix(e)
}

func (p *parser) primary() (*Expr, error) {
	switch p.tok.kind {
	case tokIdent:
		if isKeyword(p.tok.text) {
			return nil, p.bad(p.tok.text)
		}
		e := Ident(p.tok.text)
		p.next()
		return e, nil
	case tokNumber:
		e := &Expr{Int: p.tok.intVal}
		if p.tok.isFloat {
			e.Kind = ExprFloat
			e.Float = p.tok.floatVal
		} else {
			e.Kind = ExprInt
		}
		p.next()
		return e, nil
	case tokString:
		e := StringLit(p.tok.text)
		p.next()
		for p.tok.kind == tokString {
			e.Text += p.tok.text
			p.next()
		}
		return e, nil
	case tokChar:
		e := CharLit(rune(p.tok.intVal))
		p.next()
		return e, nil
	default:
		return nil, p.bad("expression")
	}
}

func (p *parser) postfix(e *Expr) (*Expr, error) {
	for {
		switch {
		case p.punct("("):
			var args []*Expr
			if !p.at(")") {
				for {
					arg, err := p.expr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if !p.punct(",") {
						break
					}
				}
			}
			if !p.punct(")") {
				return nil, p.bad(")")
			}
			e = Call(e, args...)
		case p.punct("["):
			idx, err := p.expr()
			if err != nil {
				return nil, err
			}
			if !p.punct("]") {
				return nil, p.bad("]")
			}
			e = Index(e, idx)
		case p.punct("."):
			name, ok := p.ident()
			if !ok {
				return nil, p.bad("field")
			}
			e = Dot(e, name)
		case p.punct("->"):
			name, ok := p.ident()
			if !ok {
				return nil, p.bad("field")
			}
			e = Arrow(e, name)
		case p.at("++"), p.at("--"):
			return nil, p.unsupported(p.tok.text)
		default:
			return e, nil
		}
	}
}

func binOp(tok token) (Op, int, bool) {
	if tok.kind != tokPunct {
		return 0, 0, false
	}
	switch tok.text {
	case "||":
		return OpOr, 1, true
	case "&&":
		return OpAnd, 2, true
	case "|":
		return OpBitOr, 3, true
	case "^":
		return OpBitXor, 4, true
	case "&":
		return OpBitAnd, 5, true
	case "==":
		return OpEq, 6, true
	case "!=":
		return OpNe, 6, true
	case "<":
		return OpLt, 7, true
	case ">":
		return OpGt, 7, true
	case "<=":
		return OpLe, 7, true
	case ">=":
		return OpGe, 7, true
	case "<<":
		return OpShl, 8, true
	case ">>":
		return OpShr, 8, true
	case "+":
		return OpAdd, 9, true
	case "-":
		return OpSub, 9, true
	case "*":
		return OpMul, 10, true
	case "/":
		return OpDiv, 10, true
	case "%":
		return OpMod, 10, true
	default:
		return 0, 0, false
	}
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

func (p *parser) startsDecl() bool {
	return p.startsType() || p.atKeyword("typedef") || p.atKeyword("static") || p.atKeyword("extern") || p.atKeyword("inline")
}

func (p *parser) startsType() bool {
	if p.tok.kind != tokIdent {
		return false
	}
	switch p.tok.text {
	case "void", "bool", "_Bool", "char", "short", "int", "long", "float", "double",
		"signed", "unsigned", "struct", "union", "enum", "const", "volatile":
		return true
	}
	_, ok := p.typedefs[p.tok.text]
	return ok
}

func (p *parser) startsDeclarator() bool {
	if p.at("*") || p.at("(") || p.at("[") {
		return true
	}
	return p.tok.kind == tokIdent && !isKeyword(p.tok.text)
}

func (p *parser) at(s string) bool {
	return p.tok.kind == tokPunct && p.tok.text == s
}

func (p *parser) atKeyword(s string) bool {
	return p.tok.kind == tokIdent && p.tok.text == s
}

func (p *parser) punct(s string) bool {
	if p.at(s) {
		p.next()
		return true
	}
	return false
}

func (p *parser) keyword(s string) bool {
	if p.atKeyword(s) {
		p.next()
		return true
	}
	return false
}

func (p *parser) ident() (string, bool) {
	if p.tok.kind == tokIdent && !isKeyword(p.tok.text) {
		s := p.tok.text
		p.next()
		return s, true
	}
	return "", false
}

func (p *parser) bad(msg string) error {
	return fmt.Errorf("%w: line %d: %s", errSyntax, p.tok.line, msg)
}

func (p *parser) unsupported(msg string) error {
	return fmt.Errorf("%w: line %d: %s", errForm, p.tok.line, msg)
}

func isKeyword(s string) bool {
	switch s {
	case "void", "bool", "_Bool", "char", "short", "int", "long", "float", "double",
		"signed", "unsigned", "struct", "union", "enum", "typedef", "const", "volatile",
		"static", "extern", "inline", "restrict", "auto", "register",
		"if", "else", "while", "for", "return", "break", "continue", "sizeof",
		"switch", "case", "default", "do", "goto",
		"_Alignas", "_Atomic", "_Generic", "_Noreturn", "_Static_assert", "_Thread_local":
		return true
	default:
		return false
	}
}

func (p *parser) next() {
	p.skip()
	if p.i >= len(p.src) {
		p.tok = token{kind: tokEOF, line: p.line}
		return
	}
	if p.bol && p.src[p.i] == '#' {
		p.directive()
		return
	}
	p.bol = false
	p.tok = p.scan()
}

func (p *parser) skip() {
	for p.i < len(p.src) {
		c := p.src[p.i]
		switch c {
		case ' ', '\t', '\r':
			p.i++
		case '\n':
			p.i++
			p.line++
			p.bol = true
		case '/':
			if p.i+1 < len(p.src) && p.src[p.i+1] == '/' {
				p.i += 2
				for p.i < len(p.src) && p.src[p.i] != '\n' {
					p.i++
				}
				continue
			}
			if p.i+1 < len(p.src) && p.src[p.i+1] == '*' {
				p.i += 2
				for p.i+1 < len(p.src) && !(p.src[p.i] == '*' && p.src[p.i+1] == '/') {
					if p.src[p.i] == '\n' {
						p.line++
					}
					p.i++
				}
				if p.i+1 < len(p.src) {
					p.i += 2
				}
				continue
			}
			return
		default:
			return
		}
	}
}

func (p *parser) directive() {
	line := p.line
	p.i++
	p.skipHoriz()
	word := p.rawIdent()
	p.skipHoriz()
	switch word {
	case "include":
		path, system := p.headerName()
		p.finishLine()
		p.tok = token{kind: tokInclude, text: path, system: system, line: line}
	case "define":
		name := p.rawIdent()
		if p.i < len(p.src) && p.src[p.i] == '(' {
			name += "("
		}
		p.skipHoriz()
		val := p.restOfLine()
		p.tok = token{kind: tokDefine, text: name, alt: strings.TrimSpace(val), line: line}
	default:
		p.finishLine()
		p.tok = token{kind: tokPunct, text: "#" + word, line: line}
	}
	p.bol = true
}

func (p *parser) headerName() (string, bool) {
	if p.i >= len(p.src) {
		return "", false
	}
	if p.src[p.i] == '<' {
		p.i++
		start := p.i
		for p.i < len(p.src) && p.src[p.i] != '>' && p.src[p.i] != '\n' {
			p.i++
		}
		path := string(p.src[start:p.i])
		if p.i < len(p.src) && p.src[p.i] == '>' {
			p.i++
		}
		return path, true
	}
	if p.src[p.i] == '"' {
		p.i++
		start := p.i
		for p.i < len(p.src) && p.src[p.i] != '"' && p.src[p.i] != '\n' {
			p.i++
		}
		path := string(p.src[start:p.i])
		if p.i < len(p.src) && p.src[p.i] == '"' {
			p.i++
		}
		return path, false
	}
	return "", false
}

func (p *parser) rawIdent() string {
	start := p.i
	if p.i < len(p.src) && (isIdentStart(p.src[p.i])) {
		p.i++
		for p.i < len(p.src) && isIdentCont(p.src[p.i]) {
			p.i++
		}
	}
	return string(p.src[start:p.i])
}

func (p *parser) skipHoriz() {
	for p.i < len(p.src) && (p.src[p.i] == ' ' || p.src[p.i] == '\t') {
		p.i++
	}
}

func (p *parser) restOfLine() string {
	start := p.i
	for p.i < len(p.src) && p.src[p.i] != '\n' {
		p.i++
	}
	text := string(p.src[start:p.i])
	if p.i < len(p.src) && p.src[p.i] == '\n' {
		p.i++
		p.line++
	}
	return text
}

func (p *parser) finishLine() {
	for p.i < len(p.src) && p.src[p.i] != '\n' {
		p.i++
	}
	if p.i < len(p.src) && p.src[p.i] == '\n' {
		p.i++
		p.line++
	}
}

func (p *parser) scan() token {
	line := p.line
	if p.i >= len(p.src) {
		return token{kind: tokEOF, line: line}
	}
	c := p.src[p.i]
	if isIdentStart(c) {
		start := p.i
		p.i++
		for p.i < len(p.src) && isIdentCont(p.src[p.i]) {
			p.i++
		}
		return token{kind: tokIdent, text: string(p.src[start:p.i]), line: line}
	}
	if c >= '0' && c <= '9' {
		return p.scanNumber(line)
	}
	if c == '"' {
		text, ok := p.scanString(line)
		if !ok {
			return token{kind: tokPunct, text: string(c), line: line}
		}
		return token{kind: tokString, text: text, line: line}
	}
	if c == '\'' {
		val, ok := p.scanChar(line)
		if !ok {
			return token{kind: tokPunct, text: string(c), line: line}
		}
		return token{kind: tokChar, intVal: val, line: line}
	}
	if text, ok := p.scanPunct(); ok {
		return token{kind: tokPunct, text: text, line: line}
	}
	p.i++
	return token{kind: tokPunct, text: string(c), line: line}
}

func (p *parser) scanNumber(line int) token {
	start := p.i
	isFloat := false
	if p.src[p.i] == '0' && p.i+1 < len(p.src) && (p.src[p.i+1] == 'x' || p.src[p.i+1] == 'X') {
		p.i += 2
		for p.i < len(p.src) && isHex(p.src[p.i]) {
			p.i++
		}
	} else {
		for p.i < len(p.src) && p.src[p.i] >= '0' && p.src[p.i] <= '9' {
			p.i++
		}
		if p.i < len(p.src) && p.src[p.i] == '.' {
			isFloat = true
			p.i++
			for p.i < len(p.src) && p.src[p.i] >= '0' && p.src[p.i] <= '9' {
				p.i++
			}
		}
		if p.i < len(p.src) && (p.src[p.i] == 'e' || p.src[p.i] == 'E') {
			isFloat = true
			p.i++
			if p.i < len(p.src) && (p.src[p.i] == '+' || p.src[p.i] == '-') {
				p.i++
			}
			for p.i < len(p.src) && p.src[p.i] >= '0' && p.src[p.i] <= '9' {
				p.i++
			}
		}
	}
	raw := string(p.src[start:p.i])
	for p.i < len(p.src) {
		s := p.src[p.i]
		if s == 'u' || s == 'U' || s == 'l' || s == 'L' {
			p.i++
			continue
		}
		if s == 'f' || s == 'F' {
			isFloat = true
			p.i++
			continue
		}
		break
	}
	tok := token{kind: tokNumber, text: raw, line: line, isFloat: isFloat}
	if isFloat {
		v, err := strconv.ParseFloat(raw, 64)
		if err == nil {
			tok.floatVal = v
		}
	} else if len(raw) > 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
		v, err := strconv.ParseInt(raw[2:], 16, 64)
		if err == nil {
			tok.intVal = v
		}
	} else {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			tok.intVal = v
		}
	}
	return tok
}

func (p *parser) scanString(line int) (string, bool) {
	p.i++
	var b strings.Builder
	for p.i < len(p.src) {
		c := p.src[p.i]
		if c == '"' {
			p.i++
			return b.String(), true
		}
		if c == '\n' {
			return "", false
		}
		if c == '\\' {
			r, n, ok := decodeEscape(p.src[p.i:])
			if !ok {
				return "", false
			}
			b.WriteRune(r)
			p.i += n
			continue
		}
		b.WriteByte(c)
		p.i++
	}
	_ = line
	return "", false
}

func (p *parser) scanChar(line int) (int64, bool) {
	p.i++
	if p.i >= len(p.src) {
		return 0, false
	}
	var r rune
	if p.src[p.i] == '\\' {
		decoded, n, ok := decodeEscape(p.src[p.i:])
		if !ok {
			return 0, false
		}
		r = decoded
		p.i += n
	} else {
		r = rune(p.src[p.i])
		p.i++
	}
	if p.i >= len(p.src) || p.src[p.i] != '\'' {
		return 0, false
	}
	p.i++
	_ = line
	return int64(r), true
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

func (p *parser) scanPunct() (string, bool) {
	rest := string(p.src[p.i:])
	ops := []string{
		"...", "<<=", ">>=", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
		"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "->", "++", "--",
	}
	for _, op := range ops {
		if strings.HasPrefix(rest, op) {
			p.i += len(op)
			return op, true
		}
	}
	switch p.src[p.i] {
	case '(', ')', '[', ']', '{', '}', ',', ';', '*', '=', '<', '>', '!', '~', '+', '-', '&', '|', '^', '/', '%', '.', '?', ':':
		c := p.src[p.i]
		p.i++
		return string(c), true
	default:
		return "", false
	}
}

func isIdentStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isIdentCont(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
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
