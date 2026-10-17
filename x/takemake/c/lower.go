package c

import (
	"fmt"
	"strconv"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

type syn = xparse.Syn[Term, Rule]

type parsedSpec struct {
	typ     *Type
	raw     *Type
	storage Storage
	typedef bool
	inline  bool
	body    bool
}

type builder struct {
	src      []byte
	file     *File
	tags     map[string]*Type
	typedefs map[string]*Type
	consts   map[string]int64
}

func lower(src []byte, root syn) (*File, error) {
	b := &builder{
		src:      src,
		file:     New(),
		tags:     map[string]*Type{},
		typedefs: map[string]*Type{},
		consts:   map[string]int64{},
	}
	for i := 0; i < root.ChildCount(); i++ {
		if err := b.external(root.Child(i)); err != nil {
			return nil, err
		}
	}
	return b.file, nil
}

func (b *builder) syntax(n syn, msg string) error {
	return fmt.Errorf("%w: line %d: %s", errSyntax, lineAt(b.src, n.Start()), msg)
}

func (b *builder) form(n syn, msg string) error {
	return fmt.Errorf("%w: line %d: %s", errForm, lineAt(b.src, n.Start()), msg)
}

func (b *builder) external(n syn) error {
	if tm, ok := n.Term(); ok {
		switch tm {
		case TermInclude:
			path, system, ok := headerPath(n.Text())
			if !ok {
				return b.syntax(n, "include")
			}
			b.file.Decls = append(b.file.Decls, &Decl{
				Kind:    DeclInclude,
				Include: &Include{Path: path, System: system},
			})
			return nil
		case TermInvalid:
			return b.form(n, "directive")
		}
	}
	rule, ok := n.Rule()
	if !ok {
		return b.syntax(n, "declaration")
	}
	switch rule {
	case RuleDefine:
		return b.define(n)
	case RuleDeclaration:
		return b.decl(n)
	default:
		return b.syntax(n, "declaration")
	}
}

func headerPath(text string) (string, bool, bool) {
	s := strings.TrimSpace(text)
	if i := strings.Index(s, "include"); i >= 0 {
		s = strings.TrimSpace(s[i+len("include"):])
	}
	if s == "" {
		return "", false, false
	}
	switch s[0] {
	case '<':
		j := strings.IndexByte(s, '>')
		if j < 0 {
			return "", false, false
		}
		return s[1:j], true, true
	case '"':
		j := strings.IndexByte(s[1:], '"')
		if j < 0 {
			return "", false, false
		}
		return s[1 : 1+j], false, true
	default:
		return "", false, false
	}
}

func (b *builder) define(n syn) error {
	name := ""
	var ex syn
	has := false
	for _, c := range kids(n) {
		if tm, ok := c.Term(); ok && tm == TermIdentifier {
			name = c.Text()
		}
		if r, ok := c.Rule(); ok && r == RuleExpression {
			ex = c
			has = true
		}
	}
	if name == "" {
		return b.syntax(n, "define")
	}
	val := int64(0)
	if has {
		e, err := b.expr(ex)
		if err != nil {
			return err
		}
		got, ok := evalExpr(e, b.consts)
		if !ok {
			return b.syntax(ex, "constant")
		}
		val = got
	}
	b.consts[name] = val
	b.file.Decls = append(b.file.Decls, &Decl{
		Kind:  DeclConst,
		Const: &Const{Name: name, Value: val},
	})
	return nil
}

func (b *builder) decl(n syn) error {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return b.syntax(n, "declaration")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return err
	}
	if spec.typ == nil {
		return b.syntax(n, "declaration")
	}
	if spec.body && !spec.typedef && spec.raw != nil && spec.raw.home == nil && spec.raw.Tag != "" {
		b.defineTag(spec.raw)
	}
	if blk, ok := ruleChild(n, RuleBlock); ok {
		if spec.typedef {
			return b.syntax(n, "typedef")
		}
		dN, ok := ruleChild(n, RuleDeclarator)
		if !ok {
			return b.syntax(n, "declaration")
		}
		typ, name, err := b.declarator(dN, spec.typ)
		if err != nil {
			return err
		}
		if typ == nil || typ.Kind != KindFunction {
			return b.syntax(n, "declaration")
		}
		body, err := b.block(blk)
		if err != nil {
			return err
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclFunc, Func: &Func{
			Name: name, Result: typ.Elem, Params: typ.Params, Variadic: typ.Variadic,
			Storage: spec.storage, Inline: spec.inline, Body: body,
		}})
		return nil
	}
	var inits []syn
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleInitializer {
			inits = append(inits, c)
		}
	}
	if len(inits) == 0 {
		if spec.typedef {
			return b.syntax(n, "typedef")
		}
		if !isAggregate(spec.raw) {
			return b.syntax(n, "declaration")
		}
		if spec.raw.home == nil {
			define := spec.body
			decl := &TypeDecl{Type: spec.raw, Define: define}
			b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclType, Type: decl})
			if define {
				spec.raw.home = decl
			}
		}
		return nil
	}
	if spec.typedef {
		if len(inits) != 1 {
			return b.syntax(n, "typedef")
		}
		typ, name, init, err := b.oneInit(inits[0], spec.typ)
		if err != nil {
			return err
		}
		if init != nil || name == "" {
			return b.syntax(n, "typedef")
		}
		define := spec.body && sameAggregate(typ, spec.raw)
		decl := &TypeDecl{Name: name, Type: typ, Define: define}
		if define && spec.raw != nil {
			spec.raw.home = decl
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclType, Type: decl})
		b.typedefs[name] = useType(name, typ)
		return nil
	}
	if len(inits) == 1 {
		typ, name, init, err := b.oneInit(inits[0], spec.typ)
		if err != nil {
			return err
		}
		if init == nil && typ != nil && typ.Kind == KindFunction {
			b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclFunc, Func: &Func{
				Name: name, Result: typ.Elem, Params: typ.Params, Variadic: typ.Variadic,
				Storage: spec.storage, Inline: spec.inline,
			}})
			return nil
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclObject, Object: &Object{
			Name: name, Type: detachDecl(typ), Storage: spec.storage, Init: init,
		}})
		return nil
	}
	for _, in := range inits {
		typ, name, init, err := b.oneInit(in, spec.typ)
		if err != nil {
			return err
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclObject, Object: &Object{
			Name: name, Type: detachDecl(typ), Storage: spec.storage, Init: init,
		}})
	}
	return nil
}

func (b *builder) oneInit(n syn, base *Type) (*Type, string, *Expr, error) {
	d, ok := ruleChild(n, RuleDeclarator)
	if !ok {
		return nil, "", nil, b.syntax(n, "declarator")
	}
	typ, name, err := b.declarator(d, base)
	if err != nil {
		return nil, "", nil, err
	}
	ex, ok := ruleChild(n, RuleExpression)
	if !ok {
		return typ, name, nil, nil
	}
	e, err := b.expr(ex)
	if err != nil {
		return nil, "", nil, err
	}
	return typ, name, e, nil
}

func (b *builder) specs(n syn) (parsedSpec, error) {
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
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleAggregate {
			if saw || aggr != nil || named != nil {
				return parsedSpec{}, b.syntax(c, aggrWord(c))
			}
			t, defined, err := b.aggregate(c, typedef)
			if err != nil {
				return parsedSpec{}, err
			}
			aggr = t
			body = defined
			saw = true
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermConst:
			isConst = true
		case TermVolatile:
			isVolatile = true
		case TermStatic:
			if storage != 0 || typedef {
				return parsedSpec{}, b.syntax(c, "storage class")
			}
			storage = StorageStatic
		case TermExtern:
			if storage != 0 || typedef {
				return parsedSpec{}, b.syntax(c, "storage class")
			}
			storage = StorageExtern
		case TermTypedef:
			if storage != 0 || typedef {
				return parsedSpec{}, b.syntax(c, "storage class")
			}
			typedef = true
		case TermInline:
			inline = true
		case TermVoid:
			sawVoid = true
			saw = true
		case TermBool, TermAlternateBool:
			sawBool = true
			saw = true
		case TermChar:
			sawChar = true
			saw = true
		case TermShort:
			shorts++
			saw = true
		case TermInt:
			sawInt = true
			saw = true
		case TermLong:
			longs++
			saw = true
		case TermFloat:
			sawFloat = true
			saw = true
		case TermDouble:
			sawDouble = true
			saw = true
		case TermSigned:
			sawSigned = true
			saw = true
		case TermUnsigned:
			sawUnsigned = true
			saw = true
		case TermRestrict, TermAuto, TermRegister, TermNoreturn, TermThreadLocal, TermAlignas, TermAtomic, TermGeneric:
			return parsedSpec{}, b.form(c, tm.String())
		case TermTypeName:
			if saw || aggr != nil || named != nil {
				return parsedSpec{}, b.syntax(c, "type")
			}
			t := b.typedefs[c.Text()]
			if t == nil {
				return parsedSpec{}, b.syntax(c, "type")
			}
			named = t
			saw = true
		default:
			return parsedSpec{}, b.syntax(c, tm.String())
		}
	}
	spec := parsedSpec{storage: storage, typedef: typedef, inline: inline, body: body}
	switch {
	case aggr != nil:
		spec.raw = aggr
		spec.typ = qualify(aggr, isConst, isVolatile)
	case named != nil:
		cp := *named
		cp.Const = cp.Const || isConst
		cp.Volatile = cp.Volatile || isVolatile
		cp.home = nil
		cp.laying = false
		spec.typ = &cp
		spec.raw = named
	default:
		t, err := b.scalar(n, sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned, longs, shorts)
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

func (b *builder) scalar(n syn, sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned bool, longs, shorts int) (*Type, error) {
	switch {
	case sawVoid:
		return Void(), nil
	case sawBool:
		return Bool(), nil
	case sawFloat && longs > 0:
		return nil, b.syntax(n, "float")
	case sawFloat:
		return Float(), nil
	case sawDouble && longs > 0:
		return LongDouble(), nil
	case sawDouble:
		return Double(), nil
	}
	if !sawChar && !sawInt && !sawSigned && !sawUnsigned && longs == 0 && shorts == 0 {
		return nil, b.syntax(n, "type")
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

func (b *builder) aggregate(n syn, typedef bool) (*Type, bool, error) {
	kind := KindStruct
	word := "struct"
	tag := ""
	body := false
	var fields []syn
	var enums syn
	hasEnums := false
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RuleField:
				fields = append(fields, c)
			case RuleEnumeratorList:
				enums = c
				hasEnums = true
			}
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermStruct:
			kind, word = KindStruct, "struct"
		case TermUnion:
			kind, word = KindUnion, "union"
		case TermEnum:
			kind, word = KindEnum, "enum"
		case TermIdentifier:
			tag = c.Text()
		case TermLeftBrace:
			body = true
		}
	}
	if !body {
		if tag == "" {
			return nil, false, b.syntax(n, word)
		}
		return b.tag(kind, tag), false, nil
	}
	var t *Type
	if tag == "" {
		t = &Type{Kind: kind, Inline: true}
	} else {
		t = b.tag(kind, tag)
		if t.Defined {
			return nil, false, b.syntax(n, "redefinition")
		}
	}
	if kind == KindEnum {
		if hasEnums {
			if err := b.enumerators(t, enums); err != nil {
				return nil, false, err
			}
		}
	} else if err := b.fields(t, fields); err != nil {
		return nil, false, err
	}
	t.Defined = true
	if tag == "" {
		t.Inline = true
	}
	layout(t)
	if tag != "" && !typedef {
		b.defineTag(t)
	}
	return t, true, nil
}

func aggrWord(n syn) string {
	for _, c := range kids(n) {
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermStruct, TermUnion, TermEnum:
			return tm.String()
		}
	}
	return "type"
}

func (b *builder) defineTag(t *Type) {
	if t == nil || t.Tag == "" {
		return
	}
	if t.home != nil && t.home.Define {
		return
	}
	decl := &TypeDecl{Type: t, Define: true}
	t.home = decl
	t.Inline = false
	b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclType, Type: decl})
}

func (b *builder) tag(kind Kind, tag string) *Type {
	key := aggregateWord(kind) + " " + tag
	if t := b.tags[key]; t != nil {
		return t
	}
	t := &Type{Kind: kind, Tag: tag}
	if kind == KindEnum {
		t.Size = intSize
		t.Align = intSize
	}
	b.tags[key] = t
	return t
}

func (b *builder) fields(parent *Type, nodes []syn) error {
	for _, n := range nodes {
		if err := b.field(parent, n); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) field(parent *Type, n syn) error {
	for _, c := range kids(n) {
		if tm, ok := c.Term(); ok && tm == TermColon {
			return b.form(c, "bit-field")
		}
	}
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return b.syntax(n, "field")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return err
	}
	var decls []syn
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleDeclarator {
			decls = append(decls, c)
		}
	}
	if len(decls) == 0 {
		parent.Fields = append(parent.Fields, &Field{Type: detachDecl(spec.typ)})
		return nil
	}
	for _, d := range decls {
		typ, name, err := b.declarator(d, spec.typ)
		if err != nil {
			return err
		}
		parent.Fields = append(parent.Fields, &Field{Name: name, Type: detachDecl(typ)})
	}
	return nil
}

func (b *builder) enumerators(t *Type, n syn) error {
	next := int64(0)
	for _, item := range enumItems(n) {
		name, val, err := b.enumerator(item, next)
		if err != nil {
			return err
		}
		next = val + 1
		t.Enums = append(t.Enums, &Enumerator{Name: name, Value: val})
		b.consts[name] = val
	}
	return nil
}

func (b *builder) enumerator(n syn, next int64) (string, int64, error) {
	name := ""
	var ex syn
	has := false
	for _, c := range kids(n) {
		if tm, ok := c.Term(); ok && tm == TermIdentifier {
			name = c.Text()
		}
		if r, ok := c.Rule(); ok && r == RuleExpression {
			ex = c
			has = true
		}
	}
	if name == "" {
		return "", 0, b.syntax(n, "enumerator")
	}
	val := next
	if has {
		e, err := b.expr(ex)
		if err != nil {
			return "", 0, err
		}
		got, ok := evalExpr(e, b.consts)
		if !ok {
			return "", 0, b.syntax(ex, "enumerator")
		}
		val = got
	}
	return name, val, nil
}

func enumItems(n syn) []syn {
	var out []syn
	var walk func(syn)
	walk = func(n syn) {
		r, ok := n.Rule()
		if !ok {
			return
		}
		if r == RuleEnumerator {
			out = append(out, n)
			return
		}
		if r != RuleEnumeratorList {
			return
		}
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if cr, ok := c.Rule(); ok && (cr == RuleEnumeratorList || cr == RuleEnumerator) {
				walk(c)
			}
		}
	}
	walk(n)
	return out
}

type ptrQual struct {
	cst, vol, restrict bool
	at                 syn
}

type suf struct {
	array    bool
	flex     bool
	expr     syn
	fn       bool
	params   []*Param
	variadic bool
}

type dtree struct {
	ptrs  []ptrQual
	name  string
	inner *dtree
	sufs  []suf
}

func (b *builder) declarator(n syn, base *Type) (*Type, string, error) {
	d, err := b.readDecl(n)
	if err != nil {
		return nil, "", err
	}
	return b.apply(d, base)
}

func (b *builder) readDecl(n syn) (*dtree, error) {
	d := &dtree{}
	if p, ok := ruleChild(n, RulePointer); ok {
		qs, err := b.readPtrs(p)
		if err != nil {
			return nil, err
		}
		d.ptrs = qs
	}
	dir, ok := ruleChild(n, RuleDirectDeclarator)
	if !ok {
		return d, nil
	}
	name, inner, sufs, err := b.readDirect(dir)
	if err != nil {
		return nil, err
	}
	d.name = name
	d.inner = inner
	d.sufs = sufs
	return d, nil
}

func (b *builder) readPtrs(n syn) ([]ptrQual, error) {
	var qs []ptrQual
	cur := n
	for cur.Valid() {
		q := ptrQual{}
		var next syn
		for _, c := range kids(cur) {
			if tm, ok := c.Term(); ok {
				switch tm {
				case TermConst:
					q.cst = true
				case TermVolatile:
					q.vol = true
				case TermRestrict:
					q.restrict = true
					q.at = c
				case TermStar:
					if !q.at.Valid() {
						q.at = c
					}
				}
			}
			if r, ok := c.Rule(); ok && r == RulePointer {
				next = c
			}
		}
		if q.restrict {
			return nil, b.form(q.at, "restrict")
		}
		qs = append(qs, q)
		cur = next
	}
	return qs, nil
}

func (b *builder) readDirect(n syn) (string, *dtree, []suf, error) {
	var name string
	var inner *dtree
	var sufs []suf
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RuleDeclarator:
				in, err := b.readDecl(c)
				if err != nil {
					return "", nil, nil, err
				}
				inner = in
			case RuleSuffix:
				s, err := b.readSuf(c)
				if err != nil {
					return "", nil, nil, err
				}
				sufs = append(sufs, s)
			}
			continue
		}
		if tm, ok := c.Term(); ok && tm == TermIdentifier {
			name = c.Text()
		}
	}
	return name, inner, sufs, nil
}

func (b *builder) readSuf(n syn) (suf, error) {
	brack := false
	for _, c := range kids(n) {
		if tm, ok := c.Term(); ok && tm == TermLeftBracket {
			brack = true
		}
	}
	if brack {
		s := suf{array: true, flex: true}
		if e, ok := ruleChild(n, RuleExpression); ok {
			s.flex = false
			s.expr = e
		}
		return s, nil
	}
	s := suf{fn: true}
	if p, ok := ruleChild(n, RuleParameters); ok {
		params, variadic, err := b.parameters(p)
		if err != nil {
			return suf{}, err
		}
		s.params = params
		s.variadic = variadic
	}
	return s, nil
}

func (b *builder) apply(d *dtree, base *Type) (*Type, string, error) {
	if d == nil {
		return base, "", nil
	}
	t := base
	for _, q := range d.ptrs {
		t = &Type{Kind: KindPointer, Elem: t, Const: q.cst, Volatile: q.vol, Size: ptrSize, Align: ptrSize}
	}
	var err error
	for i := len(d.sufs) - 1; i >= 0; i-- {
		t, err = b.applySuf(d.sufs[i], t)
		if err != nil {
			return nil, "", err
		}
	}
	name := d.name
	if d.inner != nil {
		t, name, err = b.apply(d.inner, t)
		if err != nil {
			return nil, "", err
		}
	}
	return t, name, nil
}

func (b *builder) applySuf(s suf, t *Type) (*Type, error) {
	if s.fn {
		return &Type{Kind: KindFunction, Elem: t, Params: s.params, Variadic: s.variadic, Align: 1}, nil
	}
	length := -1
	if !s.flex {
		e, err := b.expr(s.expr)
		if err != nil {
			return nil, err
		}
		v, ok := evalExpr(e, b.consts)
		if !ok || v < 0 {
			return nil, b.syntax(s.expr, "array length")
		}
		length = int(v)
	}
	a := &Type{Kind: KindArray, Elem: t, Len: length}
	layout(a)
	return a, nil
}

func (b *builder) parameters(n syn) ([]*Param, bool, error) {
	if n.ChildCount() == 1 {
		if tm, ok := n.Child(0).Term(); ok && tm == TermEllipsis {
			return nil, true, nil
		}
	}
	pnode, ok := ruleChild(n, RuleParameter)
	if !ok {
		return nil, false, b.syntax(n, "parameter")
	}
	tail, _ := ruleChild(n, RuleParameterTail)
	var out []*Param
	variadic := false
	for {
		p, err := b.param(pnode)
		if err != nil {
			return nil, false, err
		}
		out = append(out, p)
		if !tail.Valid() || tail.ChildCount() == 0 {
			break
		}
		next, ok := ruleChild(tail, RuleParameter)
		if !ok {
			for _, c := range kids(tail) {
				if tm, ok := c.Term(); ok && tm == TermEllipsis {
					variadic = true
				}
			}
			break
		}
		pnode = next
		tail, _ = ruleChild(tail, RuleParameterTail)
	}
	if !variadic && len(out) == 1 && out[0] != nil && out[0].Name == "" && out[0].Type != nil && out[0].Type.Kind == KindVoid {
		return nil, false, nil
	}
	return out, variadic, nil
}

func (b *builder) param(n syn) (*Param, error) {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return nil, b.syntax(n, "parameter")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return nil, err
	}
	typ := spec.typ
	name := ""
	if d, ok := ruleChild(n, RuleDeclarator); ok {
		typ, name, err = b.declarator(d, spec.typ)
		if err != nil {
			return nil, err
		}
	}
	return &Param{Name: name, Type: typ}, nil
}

func (b *builder) typeName(n syn, what string) (*Type, error) {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return nil, b.syntax(n, what)
	}
	spec, err := b.specs(specN)
	if err != nil {
		return nil, err
	}
	typ := spec.typ
	if d, ok := ruleChild(n, RuleDeclarator); ok {
		var name string
		typ, name, err = b.declarator(d, spec.typ)
		if err != nil {
			return nil, err
		}
		if name != "" {
			return nil, b.syntax(d, what)
		}
	}
	if typ == nil {
		return nil, b.syntax(n, what)
	}
	return typ, nil
}

func (b *builder) block(n syn) (*Stmt, error) {
	var list []*Stmt
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleStatement {
			s, err := b.stmt(c)
			if err != nil {
				return nil, err
			}
			list = append(list, s)
		}
	}
	return Block(list...), nil
}

func (b *builder) stmt(n syn) (*Stmt, error) {
	if r, ok := n.Rule(); ok {
		switch r {
		case RuleBlock:
			return b.block(n)
		case RuleLocal:
			return b.local(n)
		case RuleStatement:
		default:
			return nil, b.syntax(n, "statement")
		}
	}
	var keyword Term
	hasKeyword := false
	var exprs []syn
	var stmts []syn
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RuleExpression:
				exprs = append(exprs, c)
			case RuleStatement, RuleBlock, RuleLocal:
				stmts = append(stmts, c)
			}
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermIf, TermWhile, TermFor, TermReturn, TermBreak, TermContinue, TermSwitch, TermDo, TermGoto:
			if !hasKeyword {
				keyword = tm
				hasKeyword = true
			}
		}
	}
	switch keyword {
	case TermSwitch:
		return nil, b.form(n, "switch")
	case TermDo:
		return nil, b.form(n, "do")
	case TermGoto:
		return nil, b.form(n, "goto")
	case TermBreak:
		return Break(), nil
	case TermContinue:
		return Continue(), nil
	case TermReturn:
		var e *Expr
		if len(exprs) > 0 {
			var err error
			e, err = b.expr(exprs[0])
			if err != nil {
				return nil, err
			}
		}
		return Return(e), nil
	case TermIf:
		if len(exprs) < 1 || len(stmts) < 1 {
			return nil, b.syntax(n, "if")
		}
		cond, err := b.expr(exprs[0])
		if err != nil {
			return nil, err
		}
		body, err := b.stmt(stmts[0])
		if err != nil {
			return nil, err
		}
		s := If(cond, body)
		if len(stmts) > 1 {
			alt, err := b.stmt(stmts[1])
			if err != nil {
				return nil, err
			}
			s.Else = alt
		}
		return s, nil
	case TermWhile:
		if len(exprs) < 1 || len(stmts) < 1 {
			return nil, b.syntax(n, "while")
		}
		cond, err := b.expr(exprs[0])
		if err != nil {
			return nil, err
		}
		body, err := b.stmt(stmts[0])
		if err != nil {
			return nil, err
		}
		return While(cond, body), nil
	case TermFor:
		return b.forStmt(n)
	}
	if len(stmts) == 1 && len(exprs) == 0 {
		return b.stmt(stmts[0])
	}
	if len(exprs) == 1 {
		e, err := b.expr(exprs[0])
		if err != nil {
			return nil, err
		}
		return ExprStmt(e), nil
	}
	return nil, b.syntax(n, "statement")
}

func (b *builder) forStmt(n syn) (*Stmt, error) {
	var init *Stmt
	var cond, post *Expr
	var body *Stmt
	seenSemi := false
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RuleForInitializer:
				s, err := b.forInit(c)
				if err != nil {
					return nil, err
				}
				init = s
			case RuleExpression:
				e, err := b.expr(c)
				if err != nil {
					return nil, err
				}
				if !seenSemi {
					cond = e
				} else {
					post = e
				}
			case RuleStatement:
				s, err := b.stmt(c)
				if err != nil {
					return nil, err
				}
				body = s
			}
			continue
		}
		if tm, ok := c.Term(); ok && tm == TermSemicolon {
			seenSemi = true
		}
	}
	return For(init, cond, post, body), nil
}

func (b *builder) forInit(n syn) (*Stmt, error) {
	if n.ChildCount() == 1 {
		if tm, ok := n.Child(0).Term(); ok && tm == TermSemicolon {
			return nil, nil
		}
	}
	for _, c := range kids(n) {
		r, ok := c.Rule()
		if !ok {
			continue
		}
		switch r {
		case RuleLocal:
			return b.local(c)
		case RuleExpression:
			e, err := b.expr(c)
			if err != nil {
				return nil, err
			}
			return ExprStmt(e), nil
		}
	}
	return nil, nil
}

func (b *builder) local(n syn) (*Stmt, error) {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return nil, b.syntax(n, "declaration")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return nil, err
	}
	if spec.typedef {
		return nil, b.form(n, "local typedef")
	}
	var objs []*Object
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleInitializer {
			typ, name, init, err := b.oneInit(c, spec.typ)
			if err != nil {
				return nil, err
			}
			objs = append(objs, &Object{Name: name, Type: typ, Storage: spec.storage, Init: init})
		}
	}
	return &Stmt{Kind: StmtDecl, Objects: objs}, nil
}

func (b *builder) expr(n syn) (*Expr, error) {
	r, ok := n.Rule()
	if !ok {
		return nil, b.syntax(n, "expression")
	}
	switch r {
	case RuleExpression:
		return b.exprExpr(n)
	case RuleUnary:
		return b.unary(n)
	case RulePrefix:
		return b.pre(n)
	case RulePostfix:
		return b.postfix(n)
	case RulePrimary:
		return b.primary(n)
	case RuleText:
		s, err := b.rawText(n)
		if err != nil {
			return nil, err
		}
		return StringLit(s), nil
	default:
		return nil, b.syntax(n, "expression")
	}
}

func (b *builder) exprExpr(n syn) (*Expr, error) {
	var op Term
	hasOp := false
	var subs []syn
	for _, c := range kids(n) {
		if _, ok := c.Rule(); ok {
			subs = append(subs, c)
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		if tm == TermEqual || compoundOp(tm) {
			op = tm
			hasOp = true
			continue
		}
		if _, ok := binaryOp(tm); ok {
			op = tm
			hasOp = true
		}
	}
	if hasOp && len(subs) == 2 {
		left, err := b.expr(subs[0])
		if err != nil {
			return nil, err
		}
		right, err := b.expr(subs[1])
		if err != nil {
			return nil, err
		}
		if op == TermEqual {
			return Assign(left, right), nil
		}
		if compoundOp(op) {
			return nil, b.form(n, "assignment")
		}
		bop, ok := binaryOp(op)
		if !ok {
			return nil, b.syntax(n, "expression")
		}
		return Binary(bop, left, right), nil
	}
	if len(subs) == 1 && !hasOp {
		return b.expr(subs[0])
	}
	return nil, b.syntax(n, "expression")
}

func (b *builder) unary(n syn) (*Expr, error) {
	var typ syn
	hasTyp := false
	var sub syn
	for _, c := range kids(n) {
		r, ok := c.Rule()
		if !ok {
			continue
		}
		if r == RuleTypeName {
			typ = c
			hasTyp = true
			continue
		}
		sub = c
	}
	if hasTyp {
		t, err := b.typeName(typ, "cast")
		if err != nil {
			return nil, err
		}
		x, err := b.expr(sub)
		if err != nil {
			return nil, err
		}
		return Cast(t, x), nil
	}
	if sub.Valid() {
		return b.expr(sub)
	}
	return nil, b.syntax(n, "expression")
}

func (b *builder) pre(n syn) (*Expr, error) {
	var op Term
	hasOp := false
	var sub syn
	hasSub := false
	var typ syn
	hasTyp := false
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			if r == RuleTypeName {
				typ = c
				hasTyp = true
				continue
			}
			sub = c
			hasSub = true
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermLeftParenthesis, TermRightParenthesis:
		default:
			op = tm
			hasOp = true
		}
	}
	if hasOp && op == TermSizeof && hasTyp {
		t, err := b.typeName(typ, "sizeof")
		if err != nil {
			return nil, err
		}
		return SizeofType(t), nil
	}
	if hasOp && op == TermSizeof && hasSub {
		x, err := b.expr(sub)
		if err != nil {
			return nil, err
		}
		return SizeofExpr(x), nil
	}
	if hasOp && (op == TermIncrement || op == TermDecrement) {
		return nil, b.form(n, op.String())
	}
	if hasOp && hasSub {
		pop, ok := prefixOp(op)
		if !ok {
			return nil, b.syntax(n, "expression")
		}
		x, err := b.expr(sub)
		if err != nil {
			return nil, err
		}
		return Unary(pop, x), nil
	}
	if hasSub {
		return b.expr(sub)
	}
	return nil, b.syntax(n, "expression")
}

func (b *builder) postfix(n syn) (*Expr, error) {
	var base syn
	var idx syn
	var argsN syn
	hasArgs := false
	field := ""
	brack := false
	call := false
	dot := false
	arrow := false
	inc := false
	dec := false
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RulePostfix, RulePrimary:
				if !base.Valid() {
					base = c
				}
			case RuleExpression:
				idx = c
			case RuleArguments:
				argsN = c
				hasArgs = true
			}
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermLeftBracket:
			brack = true
		case TermLeftParenthesis:
			call = true
		case TermDot:
			dot = true
		case TermArrow:
			arrow = true
		case TermIdentifier:
			field = c.Text()
		case TermIncrement:
			inc = true
		case TermDecrement:
			dec = true
		}
	}
	if inc {
		return nil, b.form(n, "++")
	}
	if dec {
		return nil, b.form(n, "--")
	}
	if !base.Valid() {
		return nil, b.syntax(n, "expression")
	}
	left, err := b.expr(base)
	if err != nil {
		return nil, err
	}
	switch {
	case brack:
		i, err := b.expr(idx)
		if err != nil {
			return nil, err
		}
		return Index(left, i), nil
	case call:
		var list []*Expr
		if hasArgs {
			list, err = b.args(argsN)
			if err != nil {
				return nil, err
			}
		}
		return Call(left, list...), nil
	case dot:
		return Dot(left, field), nil
	case arrow:
		return Arrow(left, field), nil
	default:
		return left, nil
	}
}

func (b *builder) args(n syn) ([]*Expr, error) {
	var out []*Expr
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleExpression {
			e, err := b.expr(c)
			if err != nil {
				return nil, err
			}
			out = append(out, e)
		}
	}
	return out, nil
}

func (b *builder) primary(n syn) (*Expr, error) {
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			switch r {
			case RuleText:
				s, err := b.rawText(c)
				if err != nil {
					return nil, err
				}
				return StringLit(s), nil
			case RuleExpression:
				return b.expr(c)
			}
			continue
		}
		tm, ok := c.Term()
		if !ok {
			continue
		}
		switch tm {
		case TermIdentifier:
			return Ident(c.Text()), nil
		case TermNumber:
			return numberExpr(c.Text()), nil
		case TermCharacterLiteral:
			v, err := charValue(c.Text())
			if err != nil {
				return nil, b.syntax(c, "char")
			}
			return CharLit(rune(v)), nil
		}
	}
	return nil, b.syntax(n, "expression")
}

func (b *builder) rawText(n syn) (string, error) {
	var bld strings.Builder
	var walk func(syn) error
	walk = func(n syn) error {
		if r, ok := n.Rule(); ok && r == RuleText {
			for _, c := range kids(n) {
				if err := walk(c); err != nil {
					return err
				}
			}
			return nil
		}
		if tm, ok := n.Term(); ok && tm == TermString {
			s, err := stringValue(n.Text())
			if err != nil {
				return b.syntax(n, "string")
			}
			bld.WriteString(s)
			return nil
		}
		return nil
	}
	if err := walk(n); err != nil {
		return "", err
	}
	return bld.String(), nil
}

func kids(n syn) []syn {
	out := make([]syn, n.ChildCount())
	for i := range out {
		out[i] = n.Child(i)
	}
	return out
}

func ruleChild(n syn, want Rule) (syn, bool) {
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if r, ok := c.Rule(); ok && r == want {
			return c, true
		}
	}
	return syn{}, false
}

func prefixOp(t Term) (Op, bool) {
	switch t {
	case TermBitwiseAnd:
		return OpAddr, true
	case TermStar:
		return OpDeref, true
	case TermLogicalNot:
		return OpNot, true
	case TermBitwiseNot:
		return OpBitNot, true
	case TermMinus:
		return OpNeg, true
	case TermPlus:
		return OpPos, true
	default:
		return 0, false
	}
}

func binaryOp(t Term) (Op, bool) {
	switch t {
	case TermLogicalOr:
		return OpOr, true
	case TermLogicalAnd:
		return OpAnd, true
	case TermBitwiseOr:
		return OpBitOr, true
	case TermBitwiseXor:
		return OpBitXor, true
	case TermBitwiseAnd:
		return OpBitAnd, true
	case TermEqualEqual:
		return OpEq, true
	case TermNotEqual:
		return OpNe, true
	case TermLess:
		return OpLt, true
	case TermGreater:
		return OpGt, true
	case TermLessEqual:
		return OpLe, true
	case TermGreaterEqual:
		return OpGe, true
	case TermShiftLeft:
		return OpShl, true
	case TermShiftRight:
		return OpShr, true
	case TermPlus:
		return OpAdd, true
	case TermMinus:
		return OpSub, true
	case TermStar:
		return OpMul, true
	case TermSlash:
		return OpDiv, true
	case TermPercent:
		return OpMod, true
	default:
		return 0, false
	}
}

func compoundOp(t Term) bool {
	switch t {
	case TermAddEqual, TermSubtractEqual, TermMultiplyEqual, TermDivideEqual, TermModuloEqual, TermAndEqual, TermOrEqual, TermXorEqual, TermShiftLeftEqual, TermShiftRightEqual:
		return true
	default:
		return false
	}
}

func numberExpr(raw string) *Expr {
	i := 0
	isFloat := false
	if len(raw) > 2 && raw[0] == '0' && (raw[1] == 'x' || raw[1] == 'X') {
		i = 2
		for i < len(raw) && isHex(raw[i]) {
			i++
		}
	} else {
		for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			i++
		}
		if i < len(raw) && raw[i] == '.' {
			isFloat = true
			i++
			for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
				i++
			}
		}
		if i < len(raw) && (raw[i] == 'e' || raw[i] == 'E') {
			isFloat = true
			i++
			if i < len(raw) && (raw[i] == '+' || raw[i] == '-') {
				i++
			}
			for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
				i++
			}
		}
	}
	body := raw[:i]
	for ; i < len(raw); i++ {
		if raw[i] == 'f' || raw[i] == 'F' {
			isFloat = true
		}
	}
	if isFloat {
		v, _ := strconv.ParseFloat(body, 64)
		return FloatLit(v)
	}
	if len(body) > 2 && body[0] == '0' && (body[1] == 'x' || body[1] == 'X') {
		v, _ := strconv.ParseInt(body[2:], 16, 64)
		return IntLit(v)
	}
	v, _ := strconv.ParseInt(body, 10, 64)
	return IntLit(v)
}

func stringValue(text string) (string, error) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", errSyntax
	}
	return decodeText([]byte(text[1 : len(text)-1]))
}

func decodeText(s []byte) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			i++
			continue
		}
		r, n, ok := decodeEscape(s[i:])
		if !ok {
			return "", errSyntax
		}
		b.WriteRune(r)
		i += n
	}
	return b.String(), nil
}

func charValue(text string) (int64, error) {
	if len(text) < 2 || text[0] != '\'' || text[len(text)-1] != '\'' {
		return 0, errSyntax
	}
	inner := text[1 : len(text)-1]
	if inner == "" {
		return 0, errSyntax
	}
	if inner[0] == '\\' {
		r, n, ok := decodeEscape([]byte(inner))
		if !ok || n != len(inner) {
			return 0, errSyntax
		}
		return int64(r), nil
	}
	if len(inner) != 1 {
		return 0, errSyntax
	}
	return int64(inner[0]), nil
}
