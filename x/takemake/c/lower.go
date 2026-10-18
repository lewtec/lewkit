package c

import (
	"fmt"
	"strconv"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

type syntaxNode = xparse.SyntaxNode[Term, Rule]

type parsedSpec struct {
	valueType *Type
	raw       *Type
	storage   Storage
	typedef   bool
	inline    bool
	body      bool
}

type builder struct {
	src      []byte
	file     *File
	tags     map[string]*Type
	typedefs map[string]*Type
	consts   map[string]int64
}

func lower(src []byte, root syntaxNode) (*File, error) {
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

func (b *builder) syntax(n syntaxNode, msg string) error {
	return fmt.Errorf("%w: line %d: %s", errSyntax, n.Line(), msg)
}

func (b *builder) form(n syntaxNode, msg string) error {
	return fmt.Errorf("%w: line %d: %s", errForm, n.Line(), msg)
}

func (b *builder) external(n syntaxNode) error {
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

func (b *builder) define(n syntaxNode) error {
	name := ""
	var ex syntaxNode
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

func (b *builder) decl(n syntaxNode) error {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return b.syntax(n, "declaration")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return err
	}
	if spec.valueType == nil {
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
		valueType, name, err := b.declarator(dN, spec.valueType)
		if err != nil {
			return err
		}
		if valueType == nil || valueType.Kind != KindFunction {
			return b.syntax(n, "declaration")
		}
		body, err := b.block(blk)
		if err != nil {
			return err
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclFunc, Func: &Func{
			Name: name, Result: valueType.Elem, Params: valueType.Params, Variadic: valueType.Variadic,
			Storage: spec.storage, Inline: spec.inline, Body: body,
		}})
		return nil
	}
	var inits []syntaxNode
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
		valueType, name, init, err := b.oneInit(inits[0], spec.valueType)
		if err != nil {
			return err
		}
		if init != nil || name == "" {
			return b.syntax(n, "typedef")
		}
		define := spec.body && sameAggregate(valueType, spec.raw)
		decl := &TypeDecl{Name: name, Type: valueType, Define: define}
		if define && spec.raw != nil {
			spec.raw.home = decl
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclType, Type: decl})
		b.typedefs[name] = useType(name, valueType)
		return nil
	}
	if len(inits) == 1 {
		valueType, name, init, err := b.oneInit(inits[0], spec.valueType)
		if err != nil {
			return err
		}
		if init == nil && valueType != nil && valueType.Kind == KindFunction {
			b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclFunc, Func: &Func{
				Name: name, Result: valueType.Elem, Params: valueType.Params, Variadic: valueType.Variadic,
				Storage: spec.storage, Inline: spec.inline,
			}})
			return nil
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclObject, Object: &Object{
			Name: name, Type: detachDecl(valueType), Storage: spec.storage, Init: init,
		}})
		return nil
	}
	for _, in := range inits {
		valueType, name, init, err := b.oneInit(in, spec.valueType)
		if err != nil {
			return err
		}
		b.file.Decls = append(b.file.Decls, &Decl{Kind: DeclObject, Object: &Object{
			Name: name, Type: detachDecl(valueType), Storage: spec.storage, Init: init,
		}})
	}
	return nil
}

func (b *builder) oneInit(n syntaxNode, base *Type) (*Type, string, *Expr, error) {
	d, ok := ruleChild(n, RuleDeclarator)
	if !ok {
		return nil, "", nil, b.syntax(n, "declarator")
	}
	valueType, name, err := b.declarator(d, base)
	if err != nil {
		return nil, "", nil, err
	}
	ex, ok := ruleChild(n, RuleExpression)
	if !ok {
		return valueType, name, nil, nil
	}
	e, err := b.expr(ex)
	if err != nil {
		return nil, "", nil, err
	}
	return valueType, name, e, nil
}

func (b *builder) specs(n syntaxNode) (parsedSpec, error) {
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
		spec.valueType = qualify(aggr, isConst, isVolatile)
	case named != nil:
		cp := *named
		cp.Const = cp.Const || isConst
		cp.Volatile = cp.Volatile || isVolatile
		cp.home = nil
		cp.laying = false
		spec.valueType = &cp
		spec.raw = named
	default:
		t, err := b.scalar(n, sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned, longs, shorts)
		if err != nil {
			return parsedSpec{}, err
		}
		t.Const = isConst
		t.Volatile = isVolatile
		spec.valueType = t
		spec.raw = t
	}
	return spec, nil
}

func (b *builder) scalar(n syntaxNode, sawVoid, sawBool, sawChar, sawInt, sawFloat, sawDouble, sawSigned, sawUnsigned bool, longs, shorts int) (*Type, error) {
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

func (b *builder) aggregate(n syntaxNode, typedef bool) (*Type, bool, error) {
	kind := KindStruct
	word := "struct"
	tag := ""
	body := false
	var fields []syntaxNode
	var enums syntaxNode
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

func aggrWord(n syntaxNode) string {
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

func (b *builder) fields(parent *Type, nodes []syntaxNode) error {
	for _, n := range nodes {
		if err := b.field(parent, n); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) field(parent *Type, n syntaxNode) error {
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
	var decls []syntaxNode
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok && r == RuleDeclarator {
			decls = append(decls, c)
		}
	}
	if len(decls) == 0 {
		parent.Fields = append(parent.Fields, &Field{Type: detachDecl(spec.valueType)})
		return nil
	}
	for _, d := range decls {
		valueType, name, err := b.declarator(d, spec.valueType)
		if err != nil {
			return err
		}
		parent.Fields = append(parent.Fields, &Field{Name: name, Type: detachDecl(valueType)})
	}
	return nil
}

func (b *builder) enumerators(t *Type, n syntaxNode) error {
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

func (b *builder) enumerator(n syntaxNode, next int64) (string, int64, error) {
	name := ""
	var ex syntaxNode
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

func enumItems(n syntaxNode) []syntaxNode {
	var out []syntaxNode
	var walk func(syntaxNode)
	walk = func(n syntaxNode) {
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

type pointerQualifier struct {
	constant, volatile, restrict bool
	at                           syntaxNode
}

type suffix struct {
	array      bool
	flexible   bool
	expression syntaxNode
	function   bool
	parameters []*Param
	variadic   bool
}

type declaratorTree struct {
	pointers []pointerQualifier
	name     string
	inner    *declaratorTree
	suffixes []suffix
}

func (b *builder) declarator(node syntaxNode, base *Type) (*Type, string, error) {
	tree, err := b.readDeclarator(node)
	if err != nil {
		return nil, "", err
	}
	return b.applyDeclarator(tree, base)
}

func (b *builder) readDeclarator(node syntaxNode) (*declaratorTree, error) {
	tree := &declaratorTree{}
	if pointer, ok := ruleChild(node, RulePointer); ok {
		qualifiers, err := b.readPointers(pointer)
		if err != nil {
			return nil, err
		}
		tree.pointers = qualifiers
	}
	direct, ok := ruleChild(node, RuleDirectDeclarator)
	if !ok {
		return tree, nil
	}
	name, inner, suffixes, err := b.readDirect(direct)
	if err != nil {
		return nil, err
	}
	tree.name = name
	tree.inner = inner
	tree.suffixes = suffixes
	return tree, nil
}

func (b *builder) readPointers(node syntaxNode) ([]pointerQualifier, error) {
	var qualifiers []pointerQualifier
	current := node
	for current.Valid() {
		qualifier := pointerQualifier{}
		var next syntaxNode
		for _, child := range kids(current) {
			if term, ok := child.Term(); ok {
				switch term {
				case TermConst:
					qualifier.constant = true
				case TermVolatile:
					qualifier.volatile = true
				case TermRestrict:
					qualifier.restrict = true
					qualifier.at = child
				case TermStar:
					if !qualifier.at.Valid() {
						qualifier.at = child
					}
				}
			}
			if rule, ok := child.Rule(); ok && rule == RulePointer {
				next = child
			}
		}
		if qualifier.restrict {
			return nil, b.form(qualifier.at, "restrict")
		}
		qualifiers = append(qualifiers, qualifier)
		current = next
	}
	return qualifiers, nil
}

func (b *builder) readDirect(node syntaxNode) (string, *declaratorTree, []suffix, error) {
	var name string
	var inner *declaratorTree
	var suffixes []suffix
	for _, child := range kids(node) {
		if rule, ok := child.Rule(); ok {
			switch rule {
			case RuleDeclarator:
				nested, err := b.readDeclarator(child)
				if err != nil {
					return "", nil, nil, err
				}
				inner = nested
			case RuleSuffix:
				one, err := b.readSuffix(child)
				if err != nil {
					return "", nil, nil, err
				}
				suffixes = append(suffixes, one)
			}
			continue
		}
		if term, ok := child.Term(); ok && term == TermIdentifier {
			name = child.Text()
		}
	}
	return name, inner, suffixes, nil
}

func (b *builder) readSuffix(node syntaxNode) (suffix, error) {
	bracket := false
	for _, child := range kids(node) {
		if term, ok := child.Term(); ok && term == TermLeftBracket {
			bracket = true
		}
	}
	if bracket {
		one := suffix{array: true, flexible: true}
		if expression, ok := ruleChild(node, RuleExpression); ok {
			one.flexible = false
			one.expression = expression
		}
		return one, nil
	}
	one := suffix{function: true}
	if parameters, ok := ruleChild(node, RuleParameters); ok {
		list, variadic, err := b.parameters(parameters)
		if err != nil {
			return suffix{}, err
		}
		one.parameters = list
		one.variadic = variadic
	}
	return one, nil
}

func (b *builder) applyDeclarator(tree *declaratorTree, base *Type) (*Type, string, error) {
	if tree == nil {
		return base, "", nil
	}
	valueType := base
	for _, qualifier := range tree.pointers {
		valueType = &Type{Kind: KindPointer, Elem: valueType, Const: qualifier.constant, Volatile: qualifier.volatile, Size: ptrSize, Align: ptrSize}
	}
	var err error
	for index := len(tree.suffixes) - 1; index >= 0; index-- {
		valueType, err = b.applySuffix(tree.suffixes[index], valueType)
		if err != nil {
			return nil, "", err
		}
	}
	name := tree.name
	if tree.inner != nil {
		valueType, name, err = b.applyDeclarator(tree.inner, valueType)
		if err != nil {
			return nil, "", err
		}
	}
	return valueType, name, nil
}

func (b *builder) applySuffix(one suffix, valueType *Type) (*Type, error) {
	if one.function {
		return &Type{Kind: KindFunction, Elem: valueType, Params: one.parameters, Variadic: one.variadic, Align: 1}, nil
	}
	length := -1
	if !one.flexible {
		expression, err := b.expr(one.expression)
		if err != nil {
			return nil, err
		}
		value, ok := evalExpr(expression, b.consts)
		if !ok || value < 0 {
			return nil, b.syntax(one.expression, "array length")
		}
		length = int(value)
	}
	array := &Type{Kind: KindArray, Elem: valueType, Len: length}
	layout(array)
	return array, nil
}

func (b *builder) parameters(n syntaxNode) ([]*Param, bool, error) {
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

func (b *builder) param(n syntaxNode) (*Param, error) {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return nil, b.syntax(n, "parameter")
	}
	spec, err := b.specs(specN)
	if err != nil {
		return nil, err
	}
	valueType := spec.valueType
	name := ""
	if d, ok := ruleChild(n, RuleDeclarator); ok {
		valueType, name, err = b.declarator(d, spec.valueType)
		if err != nil {
			return nil, err
		}
	}
	return &Param{Name: name, Type: valueType}, nil
}

func (b *builder) typeName(n syntaxNode, what string) (*Type, error) {
	specN, ok := ruleChild(n, RuleSpecifiers)
	if !ok {
		return nil, b.syntax(n, what)
	}
	spec, err := b.specs(specN)
	if err != nil {
		return nil, err
	}
	valueType := spec.valueType
	if d, ok := ruleChild(n, RuleDeclarator); ok {
		var name string
		valueType, name, err = b.declarator(d, spec.valueType)
		if err != nil {
			return nil, err
		}
		if name != "" {
			return nil, b.syntax(d, what)
		}
	}
	if valueType == nil {
		return nil, b.syntax(n, what)
	}
	return valueType, nil
}

func (b *builder) block(n syntaxNode) (*Stmt, error) {
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

func (b *builder) stmt(n syntaxNode) (*Stmt, error) {
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
	var exprs []syntaxNode
	var stmts []syntaxNode
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

func (b *builder) forStmt(n syntaxNode) (*Stmt, error) {
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

func (b *builder) forInit(n syntaxNode) (*Stmt, error) {
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

func (b *builder) local(n syntaxNode) (*Stmt, error) {
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
			valueType, name, init, err := b.oneInit(c, spec.valueType)
			if err != nil {
				return nil, err
			}
			objs = append(objs, &Object{Name: name, Type: valueType, Storage: spec.storage, Init: init})
		}
	}
	return &Stmt{Kind: StmtDecl, Objects: objs}, nil
}

func (b *builder) expr(n syntaxNode) (*Expr, error) {
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

func (b *builder) exprExpr(n syntaxNode) (*Expr, error) {
	var op Term
	hasOp := false
	var subs []syntaxNode
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

func (b *builder) unary(n syntaxNode) (*Expr, error) {
	var valueType syntaxNode
	hasTyp := false
	var sub syntaxNode
	for _, c := range kids(n) {
		r, ok := c.Rule()
		if !ok {
			continue
		}
		if r == RuleTypeName {
			valueType = c
			hasTyp = true
			continue
		}
		sub = c
	}
	if hasTyp {
		t, err := b.typeName(valueType, "cast")
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

func (b *builder) pre(n syntaxNode) (*Expr, error) {
	var op Term
	hasOp := false
	var sub syntaxNode
	hasSub := false
	var valueType syntaxNode
	hasTyp := false
	for _, c := range kids(n) {
		if r, ok := c.Rule(); ok {
			if r == RuleTypeName {
				valueType = c
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
		t, err := b.typeName(valueType, "sizeof")
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

func (b *builder) postfix(n syntaxNode) (*Expr, error) {
	var base syntaxNode
	var idx syntaxNode
	var argsN syntaxNode
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

func (b *builder) args(n syntaxNode) ([]*Expr, error) {
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

func (b *builder) primary(n syntaxNode) (*Expr, error) {
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

func (b *builder) rawText(n syntaxNode) (string, error) {
	var bld strings.Builder
	var walk func(syntaxNode) error
	walk = func(n syntaxNode) error {
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

func kids(n syntaxNode) []syntaxNode {
	out := make([]syntaxNode, n.ChildCount())
	for i := range out {
		out[i] = n.Child(i)
	}
	return out
}

func ruleChild(n syntaxNode, want Rule) (syntaxNode, bool) {
	for i := 0; i < n.ChildCount(); i++ {
		c := n.Child(i)
		if r, ok := c.Rule(); ok && r == want {
			return c, true
		}
	}
	return syntaxNode{}, false
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
