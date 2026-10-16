package glsl

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Take reads one GLSL shader.
// Stage is inferred from the text. Macros are expanded and not stored.
// A later [File.Make] prints the same file.
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader", ErrValue)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: read: %w", ErrValue, err)
	}
	pre, err := prepare(string(data))
	if err != nil {
		return nil, err
	}
	toks, err := lex(pre.source)
	if err != nil {
		return nil, err
	}
	file := New(0)
	file.Version = pre.version
	file.Profile = pre.profile
	file.Extensions = pre.extensions
	parser := &parser{toks: toks, types: map[string]bool{}}
	parser.parseFile(file)
	if parser.err != nil {
		return nil, parser.err
	}
	file.Stage = inferStage(file)
	return file, nil
}

type parser struct {
	toks  []token
	i     int
	err   error
	types map[string]bool
}

type quals struct {
	layouts   []Layout
	invariant bool
	precise   bool
	interp    Interp
	aux       Aux
	memory    []Memory
	rank      Rank
	storage   Storage
}

func (parser *parser) parseFile(file *File) {
	for parser.err == nil && parser.peek().kind != tokEOF {
		start := parser.i
		switch {
		case parser.identIs("precision"):
			file.Precisions = append(file.Precisions, parser.parsePrecision())
		case parser.identIs("struct"):
			parser.parseStruct(file)
		default:
			parser.parseGlobal(file)
		}
		if parser.err == nil && parser.i == start {
			parser.fail("unparsed token %s", parser.near())
			return
		}
	}
}

func (parser *parser) parsePrecision() Precision {
	parser.advance()
	rankText := parser.expectIdent()
	rank, err := ParseRank(rankText)
	if err != nil {
		parser.fail("precision %s", rankText)
	}
	typeName := parser.expectIdent()
	parser.expectOp(";")
	return Precision{Rank: rank, Type: typeName}
}

func (parser *parser) parseStruct(file *File) {
	parser.advance()
	name := parser.expectIdent()
	parser.expectOp("{")
	item := &Struct{Name: name}
	for parser.err == nil && !parser.op("}") && parser.peek().kind != tokEOF {
		item.Fields = append(item.Fields, parser.parseFields()...)
	}
	parser.expectOp("}")
	parser.types[name] = true
	file.Structs = append(file.Structs, item)
	if parser.op(";") {
		parser.advance()
		return
	}
	decls := parser.parseDeclarators(quals{}, Type{Name: name})
	parser.expectOp(";")
	file.Globals = append(file.Globals, decls...)
}

func (parser *parser) parseGlobal(file *File) {
	qual := parser.parseQualifiers()
	if parser.err != nil {
		return
	}
	if parser.op(";") {
		parser.advance()
		file.Globals = append(file.Globals, qual.decl())
		return
	}
	if parser.peek().kind == tokIdent && parser.look(1).text == "{" && !parser.isType(parser.peek().text) {
		parser.parseInterface(file, qual)
		return
	}
	if qual.present() && parser.peek().kind == tokIdent && parser.look(1).text == ";" && !parser.isType(parser.peek().text) {
		decl := qual.decl()
		decl.Name = parser.advance().text
		parser.advance()
		file.Globals = append(file.Globals, decl)
		return
	}
	typ := parser.parseType()
	if parser.err != nil {
		return
	}
	if parser.peek().kind == tokIdent && parser.look(1).text == "(" {
		file.Functions = append(file.Functions, parser.parseFunction(qual, typ))
		return
	}
	decls := parser.parseDeclarators(qual, typ)
	parser.expectOp(";")
	file.Globals = append(file.Globals, decls...)
}

func (parser *parser) parseInterface(file *File, qual quals) {
	decl := qual.decl()
	decl.Block = parser.expectIdent()
	parser.expectOp("{")
	for parser.err == nil && !parser.op("}") && parser.peek().kind != tokEOF {
		decl.Fields = append(decl.Fields, parser.parseFields()...)
	}
	parser.expectOp("}")
	if parser.peek().kind == tokIdent {
		decl.Instance = parser.advance().text
		decl.Sizes = parser.parseSizes()
	}
	parser.expectOp(";")
	file.Globals = append(file.Globals, decl)
}

func (parser *parser) parseFields() []*Decl {
	qual := parser.parseQualifiers()
	typ := parser.parseType()
	decls := parser.parseDeclarators(qual, typ)
	parser.expectOp(";")
	return decls
}

func (parser *parser) parseDeclarators(qual quals, typ Type) []*Decl {
	var decls []*Decl
	for parser.err == nil {
		name := parser.expectIdent()
		decl := qual.decl()
		decl.Type = withSizes(typ, parser.parseSizes())
		decl.Name = name
		if parser.op("=") {
			parser.advance()
			decl.Init = parser.parseInit()
		}
		decls = append(decls, decl)
		if !parser.op(",") {
			break
		}
		parser.advance()
	}
	return decls
}

func (parser *parser) parseFunction(qual quals, ret Type) *Func {
	if qual.storage != 0 || qual.invariant || qual.precise || qual.interp != 0 || qual.aux != 0 || len(qual.layouts) > 0 || len(qual.memory) > 0 {
		parser.fail("function qualifier")
	}
	fn := &Func{Rank: qual.rank, Return: ret, Name: parser.expectIdent()}
	parser.expectOp("(")
	fn.Params = parser.parseParams()
	parser.expectOp(")")
	fn.Return = withSizes(fn.Return, parser.parseSizes())
	if parser.op(";") {
		parser.advance()
		return fn
	}
	fn.Body = parser.parseBrace()
	return fn
}

func (parser *parser) parseParams() []*Param {
	if parser.op(")") {
		return nil
	}
	if parser.identIs("void") && parser.look(1).text == ")" {
		parser.advance()
		return nil
	}
	var params []*Param
	for parser.err == nil {
		params = append(params, parser.parseParam())
		if !parser.op(",") {
			break
		}
		parser.advance()
	}
	return params
}

func (parser *parser) parseParam() *Param {
	param := &Param{}
	for parser.err == nil {
		switch parser.peekIdent() {
		case "const":
			param.Const = true
			parser.advance()
			continue
		case "in", "out", "inout":
			storage, err := ParseStorage(parser.peekIdent())
			if err != nil {
				parser.fail("parameter %s", parser.peekIdent())
			}
			param.Storage = storage
			parser.advance()
			continue
		case "lowp", "mediump", "highp":
			rank, err := ParseRank(parser.peekIdent())
			if err != nil {
				parser.fail("parameter %s", parser.peekIdent())
			}
			param.Rank = rank
			parser.advance()
			continue
		case "coherent", "volatile", "restrict", "readonly", "writeonly":
			mem, err := ParseMemory(parser.peekIdent())
			if err != nil {
				parser.fail("parameter %s", parser.peekIdent())
			}
			param.Memory = addMemory(param.Memory, mem)
			parser.advance()
			continue
		default:
		}
		break
	}
	param.Type = parser.parseType()
	if parser.peek().kind == tokIdent {
		param.Name = parser.advance().text
		param.Type = withSizes(param.Type, parser.parseSizes())
	}
	return param
}

func (parser *parser) parseQualifiers() quals {
	var qual quals
	for parser.err == nil {
		switch parser.peekIdent() {
		case "layout":
			qual.layouts = append(qual.layouts, parser.parseLayout()...)
		case "invariant":
			qual.invariant = true
			parser.advance()
		case "precise":
			qual.precise = true
			parser.advance()
		case "smooth", "flat", "noperspective":
			interp, err := ParseInterp(parser.peekIdent())
			if err != nil || (qual.interp != 0 && qual.interp != interp) {
				parser.fail("interpolation %s", parser.peekIdent())
				return qual
			}
			qual.interp = interp
			parser.advance()
		case "centroid", "sample", "patch":
			aux, err := ParseAux(parser.peekIdent())
			if err != nil || (qual.aux != 0 && qual.aux != aux) {
				parser.fail("auxiliary %s", parser.peekIdent())
				return qual
			}
			qual.aux = aux
			parser.advance()
		case "coherent", "volatile", "restrict", "readonly", "writeonly":
			mem, err := ParseMemory(parser.peekIdent())
			if err != nil {
				parser.fail("memory %s", parser.peekIdent())
				return qual
			}
			qual.memory = addMemory(qual.memory, mem)
			parser.advance()
		case "lowp", "mediump", "highp":
			rank, err := ParseRank(parser.peekIdent())
			if err != nil || (qual.rank != 0 && qual.rank != rank) {
				parser.fail("precision %s", parser.peekIdent())
				return qual
			}
			qual.rank = rank
			parser.advance()
		case "const", "in", "out", "inout", "uniform", "buffer", "shared":
			storage, err := ParseStorage(parser.peekIdent())
			if err != nil || (qual.storage != 0 && qual.storage != storage) {
				parser.fail("storage %s", parser.peekIdent())
				return qual
			}
			qual.storage = storage
			parser.advance()
		default:
			return qual
		}
	}
	return qual
}

func (parser *parser) parseLayout() []Layout {
	parser.advance()
	parser.expectOp("(")
	if parser.op(")") {
		parser.advance()
		return nil
	}
	var items []Layout
	for parser.err == nil {
		item := Layout{Name: parser.expectIdent()}
		if parser.op("=") {
			parser.advance()
			item.Value = parser.parseNoComma()
		}
		items = append(items, item)
		if !parser.op(",") {
			break
		}
		parser.advance()
	}
	parser.expectOp(")")
	return items
}

func (parser *parser) parseType() Type {
	return Type{Name: parser.expectIdent(), Sizes: parser.parseSizes()}
}

func (parser *parser) parseSizes() []Size {
	var sizes []Size
	for parser.op("[") {
		parser.advance()
		if parser.op("]") {
			parser.advance()
			sizes = append(sizes, Size{})
			continue
		}
		expr := parser.parseNoComma()
		parser.expectOp("]")
		sizes = append(sizes, Size{Expr: expr})
	}
	if len(sizes) == 0 {
		return nil
	}
	return sizes
}

func (parser *parser) parseStmt() Stmt {
	if parser.err != nil {
		return nil
	}
	start := parser.i
	var stmt Stmt
	switch {
	case parser.op(";"):
		parser.advance()
		stmt = &Empty{}
	case parser.op("{"):
		stmt = &Block{Body: parser.parseBrace()}
	case parser.identIs("if"):
		stmt = parser.parseIf()
	case parser.identIs("for"):
		stmt = parser.parseFor()
	case parser.identIs("while"):
		stmt = parser.parseWhile()
	case parser.identIs("do"):
		stmt = parser.parseDo()
	case parser.identIs("switch"):
		stmt = parser.parseSwitch()
	case parser.identIs("return"):
		stmt = parser.parseReturn()
	case parser.identIs("break"):
		parser.advance()
		parser.expectOp(";")
		stmt = &Break{}
	case parser.identIs("continue"):
		parser.advance()
		parser.expectOp(";")
		stmt = &Continue{}
	case parser.identIs("discard"):
		parser.advance()
		parser.expectOp(";")
		stmt = &Discard{}
	case parser.identIs("struct"):
		parser.fail("local struct")
	case parser.startsDecl():
		stmt = parser.parseLocal()
	default:
		expr := parser.parseExpr()
		parser.expectOp(";")
		stmt = &ExprStmt{Expr: expr}
	}
	if parser.err != nil && parser.i == start && parser.peek().kind != tokEOF {
		parser.advance()
	}
	return stmt
}

func (parser *parser) parseLocal() Stmt {
	qual := parser.parseQualifiers()
	typ := parser.parseType()
	decls := parser.parseDeclarators(qual, typ)
	parser.expectOp(";")
	if len(decls) == 1 {
		return decls[0]
	}
	body := make([]Stmt, len(decls))
	for i, decl := range decls {
		body[i] = decl
	}
	return &Block{Body: body}
}

func (parser *parser) parseControlled() []Stmt {
	if parser.op("{") {
		return parser.parseBrace()
	}
	stmt := parser.parseStmt()
	if stmt == nil {
		return []Stmt{}
	}
	return []Stmt{stmt}
}

func (parser *parser) parseBrace() []Stmt {
	parser.expectOp("{")
	var body []Stmt
	for parser.err == nil && !parser.op("}") && parser.peek().kind != tokEOF {
		body = append(body, parser.parseStmt())
	}
	parser.expectOp("}")
	if body == nil {
		return []Stmt{}
	}
	return body
}

func (parser *parser) parseIf() *If {
	parser.advance()
	parser.expectOp("(")
	cond := parser.parseExpr()
	parser.expectOp(")")
	item := &If{Cond: cond, Body: parser.parseControlled()}
	if !parser.identIs("else") {
		return item
	}
	parser.advance()
	if parser.identIs("if") {
		item.Else = []Stmt{parser.parseIf()}
		return item
	}
	item.Else = parser.parseControlled()
	return item
}

func (parser *parser) parseFor() *For {
	parser.advance()
	parser.expectOp("(")
	item := &For{}
	switch {
	case parser.op(";"):
		parser.advance()
	case parser.startsDecl():
		item.Init = parser.parseLocal()
	default:
		item.Init = &ExprStmt{Expr: parser.parseExpr()}
		parser.expectOp(";")
	}
	if !parser.op(";") {
		item.Cond = parser.parseExpr()
	}
	parser.expectOp(";")
	if !parser.op(")") {
		item.Post = parser.parseExpr()
	}
	parser.expectOp(")")
	item.Body = parser.parseControlled()
	return item
}

func (parser *parser) parseWhile() *While {
	parser.advance()
	parser.expectOp("(")
	cond := parser.parseExpr()
	parser.expectOp(")")
	return &While{Cond: cond, Body: parser.parseControlled()}
}

func (parser *parser) parseDo() *Do {
	parser.advance()
	body := parser.parseControlled()
	if parser.identIs("while") {
		parser.advance()
	} else {
		parser.fail("while")
	}
	parser.expectOp("(")
	cond := parser.parseExpr()
	parser.expectOp(")")
	parser.expectOp(";")
	return &Do{Body: body, Cond: cond}
}

func (parser *parser) parseSwitch() *Switch {
	parser.advance()
	parser.expectOp("(")
	expr := parser.parseExpr()
	parser.expectOp(")")
	parser.expectOp("{")
	item := &Switch{Expr: expr}
	var current *Case
	for parser.err == nil && !parser.op("}") && parser.peek().kind != tokEOF {
		switch {
		case parser.identIs("case"):
			parser.advance()
			label := parser.parseNoComma()
			parser.expectOp(":")
			current = &Case{Expr: label}
			item.Cases = append(item.Cases, current)
		case parser.identIs("default"):
			parser.advance()
			parser.expectOp(":")
			current = &Case{Default: true}
			item.Cases = append(item.Cases, current)
		default:
			if current == nil {
				parser.fail("case")
				return item
			}
			current.Body = append(current.Body, parser.parseStmt())
		}
	}
	parser.expectOp("}")
	return item
}

func (parser *parser) parseReturn() *Return {
	parser.advance()
	if parser.op(";") {
		parser.advance()
		return &Return{}
	}
	expr := parser.parseExpr()
	parser.expectOp(";")
	return &Return{Expr: expr}
}

func (parser *parser) startsDecl() bool {
	name := parser.peekIdent()
	if name == "" || name == "true" || name == "false" {
		return false
	}
	if qualifierWord(name) {
		return true
	}
	if parser.look(1).text == "(" {
		return false
	}
	if parser.isType(name) {
		return true
	}
	return parser.look(1).kind == tokIdent
}

func (parser *parser) isType(name string) bool {
	return builtinTypes[name] || parser.types[name]
}

func (parser *parser) parseExpr() Expr { return parser.parseComma() }

func (parser *parser) parseNoComma() Expr { return parser.parseAssign() }

func (parser *parser) parseInit() Expr {
	if parser.op("{") {
		return parser.parseList()
	}
	return parser.parseNoComma()
}

func (parser *parser) parseList() Expr {
	parser.expectOp("{")
	if parser.op("}") {
		parser.advance()
		return &List{}
	}
	var exprs []Expr
	for parser.err == nil {
		exprs = append(exprs, parser.parseInit())
		if !parser.op(",") {
			break
		}
		parser.advance()
		if parser.op("}") {
			break
		}
	}
	parser.expectOp("}")
	return &List{Exprs: exprs}
}

func (parser *parser) parseComma() Expr {
	left := parser.parseAssign()
	for parser.err == nil && parser.op(",") {
		parser.advance()
		right := parser.parseAssign()
		left = &Binary{Op: ",", Left: left, Right: right}
	}
	return left
}

func (parser *parser) parseAssign() Expr {
	left := parser.parseTernary()
	op := parser.peek().text
	if parser.peek().kind == tokOp && assignOp(op) {
		parser.advance()
		right := parser.parseAssign()
		return &Binary{Op: op, Left: left, Right: right}
	}
	return left
}

func (parser *parser) parseTernary() Expr {
	cond := parser.parseOr()
	if !parser.op("?") {
		return cond
	}
	parser.advance()
	then := parser.parseAssign()
	parser.expectOp(":")
	els := parser.parseAssign()
	return &Ternary{Cond: cond, Then: then, Else: els}
}

func (parser *parser) parseOr() Expr    { return parser.parseLeft(parser.parseXor, "||") }
func (parser *parser) parseXor() Expr   { return parser.parseLeft(parser.parseAnd, "^^") }
func (parser *parser) parseAnd() Expr   { return parser.parseLeft(parser.parseBitOr, "&&") }
func (parser *parser) parseBitOr() Expr { return parser.parseLeft(parser.parseBitXor, "|") }
func (parser *parser) parseBitXor() Expr {
	return parser.parseLeft(parser.parseBitAnd, "^")
}
func (parser *parser) parseBitAnd() Expr { return parser.parseLeft(parser.parseEq, "&") }
func (parser *parser) parseEq() Expr     { return parser.parseLeft(parser.parseRel, "==", "!=") }
func (parser *parser) parseRel() Expr {
	return parser.parseLeft(parser.parseShift, "<", ">", "<=", ">=")
}
func (parser *parser) parseShift() Expr { return parser.parseLeft(parser.parseAdd, "<<", ">>") }
func (parser *parser) parseAdd() Expr   { return parser.parseLeft(parser.parseMul, "+", "-") }
func (parser *parser) parseMul() Expr   { return parser.parseLeft(parser.parseUnary, "*", "/", "%") }

func (parser *parser) parseLeft(next func() Expr, ops ...string) Expr {
	left := next()
	for parser.err == nil && parser.peek().kind == tokOp && opOne(parser.peek().text, ops) {
		op := parser.advance().text
		right := next()
		left = &Binary{Op: op, Left: left, Right: right}
	}
	return left
}

func (parser *parser) parseUnary() Expr {
	if parser.peek().kind == tokOp {
		switch parser.peek().text {
		case "++", "--", "+", "-", "!", "~":
			op := parser.advance().text
			return &Unary{Op: op, Expr: parser.parseUnary()}
		}
	}
	return parser.parsePostfix()
}

func (parser *parser) parsePostfix() Expr {
	expr := parser.parsePrimary()
	for parser.err == nil {
		switch {
		case parser.op("["):
			parser.advance()
			index := parser.parseNoComma()
			parser.expectOp("]")
			expr = &Index{Base: expr, Index: index}
		case parser.op("."):
			parser.advance()
			expr = &Select{Expr: expr, Name: parser.expectIdent()}
		case parser.op("("):
			parser.advance()
			args := parser.parseArgs()
			parser.expectOp(")")
			expr = &Call{Callee: expr, Args: args}
		case parser.op("++"), parser.op("--"):
			expr = &Postfix{Expr: expr, Op: parser.advance().text}
		default:
			return expr
		}
	}
	return expr
}

func (parser *parser) parsePrimary() Expr {
	tok := parser.peek()
	switch tok.kind {
	case tokNumber:
		parser.advance()
		if tok.lit == nil {
			parser.fail("number")
			return &Literal{Kind: KindInt}
		}
		return tok.lit
	case tokIdent:
		parser.advance()
		switch tok.text {
		case "true":
			return &Literal{Kind: KindBool, Bool: true}
		case "false":
			return &Literal{Kind: KindBool, Bool: false}
		default:
			return &Ident{Name: tok.text}
		}
	case tokOp:
		if tok.text == "(" {
			parser.advance()
			expr := parser.parseExpr()
			parser.expectOp(")")
			return expr
		}
	}
	parser.fail("expression near %s", parser.near())
	return &Literal{Kind: KindInt}
}

func (parser *parser) parseArgs() []Expr {
	if parser.op(")") {
		return nil
	}
	var args []Expr
	for parser.err == nil {
		args = append(args, parser.parseNoComma())
		if !parser.op(",") {
			break
		}
		parser.advance()
	}
	return args
}

func (parser *parser) peek() token {
	if parser.i >= len(parser.toks) {
		return token{kind: tokEOF, line: parser.line()}
	}
	return parser.toks[parser.i]
}

func (parser *parser) look(n int) token {
	i := parser.i + n
	if i < 0 || i >= len(parser.toks) {
		return token{kind: tokEOF, line: parser.line()}
	}
	return parser.toks[i]
}

func (parser *parser) advance() token {
	tok := parser.peek()
	if parser.i < len(parser.toks) {
		parser.i++
	}
	return tok
}

func (parser *parser) op(text string) bool {
	tok := parser.peek()
	return tok.kind == tokOp && tok.text == text
}

func (parser *parser) identIs(name string) bool {
	return parser.peekIdent() == name
}

func (parser *parser) peekIdent() string {
	tok := parser.peek()
	if tok.kind != tokIdent {
		return ""
	}
	return tok.text
}

func (parser *parser) expectIdent() string {
	if parser.peek().kind == tokIdent {
		return parser.advance().text
	}
	parser.fail("expected name near %s", parser.near())
	if parser.peek().kind != tokEOF {
		parser.advance()
	}
	return ""
}

func (parser *parser) expectOp(text string) {
	if parser.op(text) {
		parser.advance()
		return
	}
	parser.fail("expected %s near %s", text, parser.near())
	if parser.peek().kind != tokEOF {
		parser.advance()
	}
}

func (parser *parser) fail(format string, args ...any) {
	if parser.err != nil {
		return
	}
	parser.err = fmt.Errorf("%w: line %d: %s", ErrSyntax, parser.line(), fmt.Sprintf(format, args...))
}

func (parser *parser) line() int {
	if parser.i < len(parser.toks) {
		return parser.toks[parser.i].line
	}
	if n := len(parser.toks); n > 0 {
		return parser.toks[n-1].line
	}
	return 1
}

func (parser *parser) near() string {
	tok := parser.peek()
	if tok.kind == tokEOF {
		return "eof"
	}
	if tok.text == "" {
		return "token"
	}
	return strconv.Quote(tok.text)
}

func (qual quals) present() bool {
	return len(qual.layouts) > 0 || qual.invariant || qual.precise || qual.interp != 0 || qual.aux != 0 || len(qual.memory) > 0 || qual.rank != 0 || qual.storage != 0
}

func (qual quals) decl() *Decl {
	decl := &Decl{
		Invariant: qual.invariant,
		Precise:   qual.precise,
		Interp:    qual.interp,
		Aux:       qual.aux,
		Rank:      qual.rank,
		Storage:   qual.storage,
	}
	if len(qual.layouts) > 0 {
		decl.Layouts = append([]Layout{}, qual.layouts...)
	}
	if len(qual.memory) > 0 {
		decl.Memory = append([]Memory{}, qual.memory...)
	}
	return decl
}

func withSizes(typ Type, extra []Size) Type {
	if len(extra) == 0 {
		if len(typ.Sizes) == 0 {
			return typ
		}
		typ.Sizes = append([]Size{}, typ.Sizes...)
		return typ
	}
	sizes := make([]Size, 0, len(typ.Sizes)+len(extra))
	sizes = append(sizes, typ.Sizes...)
	sizes = append(sizes, extra...)
	typ.Sizes = sizes
	return typ
}

func qualifierWord(name string) bool {
	switch name {
	case "layout", "invariant", "precise", "smooth", "flat", "noperspective",
		"centroid", "sample", "patch", "coherent", "volatile", "restrict",
		"readonly", "writeonly", "lowp", "mediump", "highp", "const",
		"in", "out", "inout", "uniform", "buffer", "shared":
		return true
	default:
		return false
	}
}

func assignOp(op string) bool {
	switch op {
	case "=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|=":
		return true
	default:
		return false
	}
}

func opOne(op string, ops []string) bool {
	for _, item := range ops {
		if op == item {
			return true
		}
	}
	return false
}

func inferStage(file *File) Stage {
	if file == nil {
		return 0
	}
	seen := map[string]bool{}
	var discard bool
	var local bool
	var hasOut bool
	for _, decl := range file.Globals {
		if hasLocalSize(decl) {
			local = true
		}
		if decl != nil && decl.Storage == StorageOut && (decl.Name != "" || decl.Block != "") {
			hasOut = true
		}
		noteDecl(decl, seen, &discard)
	}
	for _, item := range file.Structs {
		if item == nil {
			continue
		}
		for _, field := range item.Fields {
			noteDecl(field, seen, &discard)
		}
	}
	for _, fn := range file.Functions {
		if fn == nil {
			continue
		}
		for _, stmt := range fn.Body {
			walkStmt(stmt, seen, &discard)
		}
	}
	if local {
		return Compute
	}
	if seen["gl_Position"] || seen["gl_VertexIndex"] || seen["gl_VertexID"] || seen["gl_InstanceIndex"] || seen["gl_InstanceID"] || seen["gl_PointSize"] {
		return Vertex
	}
	if discard || seen["gl_FragCoord"] || seen["gl_FrontFacing"] || seen["gl_FragDepth"] || seen["gl_PointCoord"] || seen["gl_SampleID"] {
		return Fragment
	}
	if hasOut {
		return Fragment
	}
	return 0
}

func hasLocalSize(decl *Decl) bool {
	if decl == nil {
		return false
	}
	for _, item := range decl.Layouts {
		if strings.HasPrefix(item.Name, "local_size") {
			return true
		}
	}
	for _, field := range decl.Fields {
		if hasLocalSize(field) {
			return true
		}
	}
	return false
}

func noteDecl(decl *Decl, seen map[string]bool, discard *bool) {
	if decl == nil {
		return
	}
	if decl.Name != "" {
		seen[decl.Name] = true
	}
	if decl.Instance != "" {
		seen[decl.Instance] = true
	}
	for _, item := range decl.Layouts {
		walkExpr(item.Value, seen, discard)
	}
	for _, size := range decl.Sizes {
		walkExpr(size.Expr, seen, discard)
	}
	for _, size := range decl.Type.Sizes {
		walkExpr(size.Expr, seen, discard)
	}
	walkExpr(decl.Init, seen, discard)
	for _, field := range decl.Fields {
		noteDecl(field, seen, discard)
	}
}

func walkStmt(stmt Stmt, seen map[string]bool, discard *bool) {
	if stmt == nil {
		return
	}
	switch stmt := stmt.(type) {
	case *Decl:
		noteDecl(stmt, seen, discard)
	case *Block:
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *If:
		walkExpr(stmt.Cond, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
		for _, item := range stmt.Else {
			walkStmt(item, seen, discard)
		}
	case *For:
		walkStmt(stmt.Init, seen, discard)
		walkExpr(stmt.Cond, seen, discard)
		walkExpr(stmt.Post, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *While:
		walkExpr(stmt.Cond, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *Do:
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
		walkExpr(stmt.Cond, seen, discard)
	case *Switch:
		walkExpr(stmt.Expr, seen, discard)
		for _, item := range stmt.Cases {
			if item == nil {
				continue
			}
			walkExpr(item.Expr, seen, discard)
			for _, body := range item.Body {
				walkStmt(body, seen, discard)
			}
		}
	case *Return:
		walkExpr(stmt.Expr, seen, discard)
	case *Discard:
		*discard = true
	case *ExprStmt:
		walkExpr(stmt.Expr, seen, discard)
	}
}

func walkExpr(expr Expr, seen map[string]bool, discard *bool) {
	if expr == nil {
		return
	}
	switch expr := expr.(type) {
	case *Ident:
		seen[expr.Name] = true
	case *Literal:
	case *Unary:
		walkExpr(expr.Expr, seen, discard)
	case *Binary:
		walkExpr(expr.Left, seen, discard)
		walkExpr(expr.Right, seen, discard)
	case *Ternary:
		walkExpr(expr.Cond, seen, discard)
		walkExpr(expr.Then, seen, discard)
		walkExpr(expr.Else, seen, discard)
	case *Call:
		walkExpr(expr.Callee, seen, discard)
		for _, arg := range expr.Args {
			walkExpr(arg, seen, discard)
		}
	case *Index:
		walkExpr(expr.Base, seen, discard)
		walkExpr(expr.Index, seen, discard)
	case *Select:
		walkExpr(expr.Expr, seen, discard)
		seen[expr.Name] = true
	case *Postfix:
		walkExpr(expr.Expr, seen, discard)
	case *List:
		for _, item := range expr.Exprs {
			walkExpr(item, seen, discard)
		}
	}
}

var builtinTypes = map[string]bool{}

func init() {
	addTypes("void", "bool", "int", "uint", "float", "double", "atomic_uint")
	for _, prefix := range []string{"", "i", "u", "b", "d"} {
		for _, size := range []string{"2", "3", "4"} {
			builtinTypes[prefix+"vec"+size] = true
		}
	}
	for _, size := range []string{"2", "3", "4"} {
		builtinTypes["mat"+size] = true
		builtinTypes["dmat"+size] = true
		for _, cols := range []string{"2", "3", "4"} {
			builtinTypes["mat"+size+"x"+cols] = true
			builtinTypes["dmat"+size+"x"+cols] = true
		}
	}
	for _, prefix := range []string{"", "i", "u"} {
		for _, name := range []string{
			"sampler1D", "sampler2D", "sampler3D", "samplerCube", "sampler2DRect",
			"sampler1DArray", "sampler2DArray", "samplerCubeArray", "samplerBuffer",
			"sampler2DMS", "sampler2DMSArray", "sampler1DShadow", "sampler2DShadow",
			"samplerCubeShadow", "sampler2DRectShadow", "sampler1DArrayShadow",
			"sampler2DArrayShadow", "samplerCubeArrayShadow",
			"image1D", "image2D", "image3D", "imageCube", "image2DRect",
			"image1DArray", "image2DArray", "imageCubeArray", "imageBuffer",
			"image2DMS", "image2DMSArray",
		} {
			builtinTypes[prefix+name] = true
		}
	}
	addTypes("subpassInput", "isubpassInput", "usubpassInput", "subpassInputMS", "isubpassInputMS", "usubpassInputMS")
}

func addTypes(names ...string) {
	for _, name := range names {
		builtinTypes[name] = true
	}
}
