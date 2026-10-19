package golang

import "github.com/lewtec/lewkit/x/parse"

// tokenNames are go/token strings the scanner can return, except EOF,
// COMMENT, and ILLEGAL. COMMENT is skipped. ILLEGAL fails the scan.
var tokenNames = []string{
	"IDENT", "INT", "FLOAT", "IMAG", "CHAR", "STRING",
	"+", "-", "*", "/", "%", "&", "|", "^", "<<", ">>", "&^",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^=",
	"&&", "||", "<-", "++", "--",
	"==", "<", ">", "=", "!", "!=", "<=", ">=", ":=", "...",
	"(", "[", "{", ",", ".", ")", "]", "}", ";", ":",
	"break", "default", "func", "interface", "select",
	"case", "defer", "go", "map", "struct",
	"chan", "else", "goto", "package", "switch",
	"const", "fallthrough", "if", "range", "type",
	"continue", "for", "import", "return", "var",
	"~",
}

type bld struct {
	g          *parse.Grammar
	t          map[string]*parse.Symbol
	expr       *parse.Symbol
	cond       *parse.Symbol
	typeSyntax *parse.Symbol
	block      *parse.Symbol
	sig        *parse.Symbol
}

func (b *bld) tok(name string) *parse.Symbol { return b.t[name] }

func define() *parse.Grammar {
	g := parse.New("go")
	g.Scanner(scan)
	b := &bld{g: g, t: map[string]*parse.Symbol{}}
	for _, name := range tokenNames {
		b.t[name] = g.Token(name)
	}
	b.expr = b.expressions("expr", true)
	b.cond = b.expressions("cond", false)
	b.types()
	b.signature()
	b.statements()
	b.declarations()
	b.file()
	g.Start("file")
	return g
}

func (b *bld) binops() []struct {
	op   string
	prec int
} {
	return []struct {
		op   string
		prec int
	}{
		{"||", 1},
		{"&&", 2},
		{"==", 3}, {"!=", 3}, {"<", 3}, {"<=", 3}, {">", 3}, {">=", 3},
		{"+", 4}, {"-", 4}, {"|", 4}, {"^", 4},
		{"*", 5}, {"/", 5}, {"%", 5}, {"<<", 5}, {">>", 5}, {"&", 5}, {"&^", 5},
	}
}

func (b *bld) expressions(name string, composite bool) *parse.Symbol {
	g := b.g
	e := g.Rule(name)
	for _, op := range b.binops() {
		e.Seq(g.Sym(name), b.tok(op.op), g.Sym(name)).Left(op.prec)
	}
	for _, op := range []string{"+", "-", "!", "^", "*", "&", "<-"} {
		e.Seq(b.tok(op), g.Sym(name)).Prec(6)
	}
	// Types that are not names: []byte(s), chan int(nil), make(chan int).
	e.Seq(g.Sym("type_expression"))
	if composite {
		e.Seq(g.Sym("literal_type"), b.tok("{"), g.Sym("elems"), b.tok("}")).Prec(-1)
		// T{1}, pkg.T{1}, a.b.c{1}, and T[int]{1}. The brace binds here,
		// inside an expression. if/for/switch conditions use cond, which
		// has no name composite, so the brace stays the block.
		e.Seq(g.Sym(name), b.tok("{"), g.Sym("elems"), b.tok("}")).Prec(-1)
	} else {
		// []int{1} is a condition. T{1} is not: the brace is the block.
		e.Seq(g.Sym("composite_type"), b.tok("{"), g.Sym("elems"), b.tok("}"))
	}
	e.Seq(g.Sym(name), b.tok("."), b.tok("IDENT"))
	e.Seq(g.Sym(name), b.tok("."), b.tok("("), g.Sym("type_syntax"), b.tok(")"))
	e.Seq(g.Sym(name), b.tok("("), g.Sym("args"), b.tok(")"))
	e.Seq(g.Sym(name), b.tok("["), g.Sym("brackets"))
	e.Seq(b.tok("IDENT"))
	for _, lit := range []string{"INT", "FLOAT", "IMAG", "CHAR", "STRING"} {
		e.Seq(b.tok(lit))
	}
	// Parentheses re-enter the full expression, so a composite literal is
	// legal inside them and illegal as a bare if/for/switch condition.
	e.Seq(b.tok("("), g.Sym("expr"), b.tok(")"))
	e.Seq(g.Sym("funcLit"))
	return g.Sym(name)
}

func (b *bld) types() {
	g := b.g
	expr := g.Sym("expr")
	names := g.Sep1(b.tok("IDENT"), b.tok(","))

	typeArguments := g.Rule("type_arguments")
	typeArguments.Seq(b.tok("["), g.Sep1(g.Sym("type_syntax"), b.tok(",")), b.tok("]"))
	typeArguments.Seq(b.tok("["), g.Sep1(g.Sym("type_syntax"), b.tok(",")), b.tok(","), b.tok("]"))

	// composite_type is a composite-literal type that does not start with a name.
	// In a condition it may take a brace. A name may not: if T{ is a block.
	compositeType := g.Rule("composite_type")
	compositeType.Seq(b.tok("map"), b.tok("["), g.Sym("type_syntax"), b.tok("]"), g.Sym("type_syntax"))
	compositeType.Seq(b.tok("struct"), b.tok("{"), b.semiBody("fields", g.Sym("field")), b.tok("}"))
	compositeType.Seq(b.tok("["), b.tok("]"), g.Sym("type_syntax"))
	compositeType.Seq(b.tok("["), b.tok("..."), b.tok("]"), g.Sym("type_syntax"))
	compositeType.Seq(b.tok("["), expr, b.tok("]"), g.Sym("type_syntax"))

	// type_expression is a type that can appear where an expression can, because
	// conversions and make/new take types as operands. Names stay on expr.
	typeExpression := g.Rule("type_expression")
	typeExpression.Seq(g.Sym("composite_type"))
	typeExpression.Seq(b.tok("func"), g.Sym("sig"))
	typeExpression.Seq(b.tok("interface"), b.tok("{"), b.semiBody("interface_elements", g.Sym("interface_element")), b.tok("}"))
	typeExpression.Seq(b.tok("<-"), b.tok("chan"), g.Sym("type_syntax"))
	typeExpression.Seq(b.tok("chan"), b.tok("<-"), g.Sym("type_syntax")).Dynamic(1)
	typeExpression.Seq(b.tok("chan"), g.Sym("type_syntax"))

	simple := g.Rule("simple_type")
	simple.Seq(b.tok("IDENT"))
	simple.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"))
	simple.Seq(b.tok("IDENT"), g.Sym("type_arguments"))
	simple.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"), g.Sym("type_arguments"))
	simple.Seq(b.tok("*"), g.Sym("type_syntax"))
	simple.Seq(g.Sym("type_expression"))

	typeSyntax := g.Rule("type_syntax")
	typeSyntax.Seq(g.Sym("simple_type"))
	typeSyntax.Seq(b.tok("("), g.Sym("type_syntax"), b.tok(")"))
	b.typeSyntax = g.Sym("type_syntax")

	literalType := g.Rule("literal_type")
	literalType.Seq(b.tok("IDENT"))
	literalType.Seq(b.tok("IDENT"), g.Sym("type_arguments"))
	literalType.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"))
	literalType.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"), g.Sym("type_arguments"))
	literalType.Seq(g.Sym("composite_type"))

	val := g.Rule("value")
	val.Seq(expr)
	val.Seq(b.tok("{"), g.Sym("elems"), b.tok("}"))
	el := g.Rule("elem")
	el.Seq(g.Sym("value"))
	el.Seq(g.Sym("value"), b.tok(":"), g.Sym("value"))
	elems := g.Rule("elems")
	one := g.Sep1(g.Sym("elem"), b.tok(","))
	elems.Seq()
	elems.Seq(one)
	elems.Seq(one, b.tok(","))

	field := g.Rule("field")
	field.Seq(b.tok("*"), g.Sym("embed"), b.tok("STRING"))
	field.Seq(b.tok("*"), g.Sym("embed"))
	field.Seq(names, g.Sym("type_syntax"), b.tok("STRING"))
	field.Seq(names, g.Sym("type_syntax"))
	field.Seq(g.Sym("embed"), b.tok("STRING"))
	field.Seq(g.Sym("embed"))
	embed := g.Rule("embed")
	embed.Seq(b.tok("IDENT"))
	embed.Seq(b.tok("IDENT"), g.Sym("type_arguments"))
	embed.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"))
	embed.Seq(b.tok("IDENT"), b.tok("."), b.tok("IDENT"), g.Sym("type_arguments"))

	typeElement := g.Rule("type_element")
	typeElement.Seq(g.Sym("type_syntax"))
	typeElement.Seq(b.tok("~"), g.Sym("type_syntax"))
	typeUnion := g.Rule("union")
	typeUnion.Seq(g.Sym("type_element"))
	typeUnion.Seq(g.Sym("union"), b.tok("|"), g.Sym("type_element"))

	interfaceElement := g.Rule("interface_element")
	interfaceElement.Seq(b.tok("IDENT"), g.Sym("sig"))
	interfaceElement.Seq(g.Sym("union"))

	b.brackets(expr)
	b.arguments(expr)
}

func (b *bld) brackets(expr *parse.Symbol) {
	list := b.g.Sep1(expr, b.tok(","))
	br := b.g.Rule("brackets")
	br.Seq(expr, b.tok("]"))
	br.Seq(list, b.tok(","), expr, b.tok("]"))
	br.Seq(b.tok(":"), b.tok("]"))
	br.Seq(b.tok(":"), expr, b.tok("]"))
	br.Seq(expr, b.tok(":"), b.tok("]"))
	br.Seq(expr, b.tok(":"), expr, b.tok("]"))
	br.Seq(b.tok(":"), expr, b.tok(":"), expr, b.tok("]"))
	br.Seq(expr, b.tok(":"), expr, b.tok(":"), expr, b.tok("]"))
}

func (b *bld) arguments(expr *parse.Symbol) {
	one := b.g.Sep1(expr, b.tok(","))
	args := b.g.Rule("args")
	args.Seq()
	args.Seq(one)
	args.Seq(one, b.tok(","))
	args.Seq(expr, b.tok("..."))
	args.Seq(expr, b.tok("..."), b.tok(","))
	args.Seq(one, b.tok(","), expr, b.tok("..."))
	args.Seq(one, b.tok(","), expr, b.tok("..."), b.tok(","))
}

func (b *bld) signature() {
	g := b.g
	params := g.Rule("params")
	params.Seq(b.tok("("), g.Sym("paramBody"), b.tok(")"))
	body := g.Rule("paramBody")
	body.Seq()
	body.Seq(g.Sym("paramList"))
	body.Seq(g.Sym("paramList"), b.tok(","))
	list := g.Rule("paramList")
	list.Seq(g.Sym("param"))
	list.Seq(g.Sym("paramList"), b.tok(","), g.Sym("param"))

	// A bare name reduces only before ',' ')' or ']'. A following type
	// is shifted. That keeps (int, string) and (a, b int) on one stack.
	after := g.Rule("paramAfter")
	after.Seq()
	after.Seq(b.tok("."), b.tok("IDENT"))
	after.Seq(b.tok("."), b.tok("IDENT"), g.Sym("type_arguments"))
	after.Seq(g.Sym("type_arguments"))
	after.Seq(g.Sym("type_syntax"))
	param := g.Rule("param")
	param.Seq(b.tok("..."), g.Sym("type_syntax"))
	param.Seq(b.tok("IDENT"), b.tok("..."), g.Sym("type_syntax"))
	param.Seq(b.tok("IDENT"), g.Sym("paramAfter"))
	param.Seq(b.tok("*"), g.Sym("type_syntax"))
	param.Seq(g.Sym("type_expression"))
	param.Seq(b.tok("("), g.Sym("type_syntax"), b.tok(")"))

	typeParameters := g.Rule("type_parameters")
	typeParameters.Seq(b.tok("["), g.Sym("type_parameter_list"), b.tok("]"))
	typeParameters.Seq(b.tok("["), g.Sym("type_parameter_list"), b.tok(","), b.tok("]"))
	typeParameterList := g.Rule("type_parameter_list")
	typeParameterList.Seq(g.Sym("type_parameter"))
	typeParameterList.Seq(g.Sym("type_parameter_list"), b.tok(","), g.Sym("type_parameter"))
	typeParameter := g.Rule("type_parameter")
	typeParameter.Seq(b.tok("IDENT"), g.Sym("type_parameter_tail"))
	typeParameterTail := g.Rule("type_parameter_tail")
	typeParameterTail.Seq()
	typeParameterTail.Seq(g.Sym("union"))

	result := g.Rule("result")
	result.Seq(g.Sym("params"))
	result.Seq(g.Sym("simple_type"))
	sig := g.Rule("sig")
	sig.Seq(g.Sym("params"))
	sig.Seq(g.Sym("params"), g.Sym("result"))
	b.sig = g.Sym("sig")

	fn := g.Rule("funcLit")
	fn.Seq(b.tok("func"), g.Sym("sig"), g.Sym("block"))
}

// semiBody is (elem ';')* elem? so a semicolon is required between items
// and may be absent before the closing delimiter that follows this symbol.
func (b *bld) semiBody(name string, elem *parse.Symbol) *parse.Symbol {
	rec := name + "Rec"
	r := b.g.Rule(rec)
	r.Seq()
	r.Seq(b.g.Sym(rec), elem, b.tok(";"))
	body := b.g.Rule(name)
	body.Seq(b.g.Sym(rec))
	body.Seq(b.g.Sym(rec), elem)
	return b.g.Sym(name)
}

func (b *bld) statements() {
	g := b.g
	expr := g.Sym("expr")
	cond := g.Sym("cond")
	list := g.Sep1(expr, b.tok(","))

	simple := g.Rule("simpleStmt")
	simple.Seq(list, b.tok(":="), list)
	for _, op := range []string{"=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^="} {
		simple.Seq(list, b.tok(op), list)
	}
	simple.Seq(expr, b.tok("++"))
	simple.Seq(expr, b.tok("--"))
	simple.Seq(expr, b.tok("<-"), expr)
	simple.Seq(expr)

	// plain is a statement that is not a label and not an empty statement.
	// Labels own the semicolon of their inner statement, and a label may
	// sit immediately before '}'.
	b.stmtLists()
	plain := g.Rule("plain")

	block := g.Rule("block")
	block.Seq(b.tok("{"), g.Sym("$stmtList"), b.tok("}"))
	b.block = g.Sym("block")
	plain.Seq(g.Sym("block"))

	elseOpt := g.Rule("elseOpt")
	elseOpt.Seq()
	elseOpt.Seq(b.tok("else"), g.Sym("ifStmt"))
	elseOpt.Seq(b.tok("else"), g.Sym("block"))
	ifs := g.Rule("ifStmt")
	ifs.Seq(b.tok("if"), g.Sym("simpleStmt"), b.tok(";"), cond, g.Sym("block"), g.Sym("elseOpt"))
	ifs.Seq(b.tok("if"), cond, g.Sym("block"), g.Sym("elseOpt"))

	clause := g.Rule("forClause")
	clause.Seq(b.tok(";"), g.Opt(cond), b.tok(";"), g.Opt(g.Sym("simpleStmt")))
	clause.Seq(g.Sym("simpleStmt"), b.tok(";"), g.Opt(cond), b.tok(";"), g.Opt(g.Sym("simpleStmt")))
	rng := g.Rule("rangeClause")
	rng.Seq(b.tok("range"), expr)
	rng.Seq(list, b.tok(":="), b.tok("range"), expr)
	rng.Seq(list, b.tok("="), b.tok("range"), expr)
	fr := g.Rule("forStmt")
	fr.Seq(b.tok("for"), g.Sym("block"))
	fr.Seq(b.tok("for"), cond, g.Sym("block"))
	fr.Seq(b.tok("for"), g.Sym("forClause"), g.Sym("block"))
	fr.Seq(b.tok("for"), g.Sym("rangeClause"), g.Sym("block"))

	b.caseForms("$caseHead", "$cases", "$innerCase", "$lastCase", func(h *parse.Rule) {
		h.Seq(b.tok("case"), list, b.tok(":"))
		h.Seq(b.tok("default"), b.tok(":"))
	})
	b.caseForms("$typeHead", "$typeCases", "$typeInner", "$typeLast", func(h *parse.Rule) {
		h.Seq(b.tok("case"), g.Sep1(g.Sym("type_syntax"), b.tok(",")), b.tok(":"))
		h.Seq(b.tok("default"), b.tok(":"))
	})

	sw := g.Rule("switchStmt")
	b.switchCases(sw, b.tok("switch"))
	b.switchCases(sw, b.tok("switch"), g.Sym("simpleStmt"), b.tok(";"))
	b.switchCases(sw, b.tok("switch"), g.Sym("simpleStmt"), b.tok(";"), cond)
	b.switchCases(sw, b.tok("switch"), cond)
	b.switchTypes(sw, b.tok("switch"), g.Sym("simpleStmt"), b.tok(";"), expr, b.tok("."), b.tok("("), b.tok("type"), b.tok(")"))
	b.switchTypes(sw, b.tok("switch"), g.Sym("simpleStmt"), b.tok(";"), b.tok("IDENT"), b.tok(":="), expr, b.tok("."), b.tok("("), b.tok("type"), b.tok(")"))
	b.switchTypes(sw, b.tok("switch"), expr, b.tok("."), b.tok("("), b.tok("type"), b.tok(")"))
	b.switchTypes(sw, b.tok("switch"), list, b.tok(":="), expr, b.tok("."), b.tok("("), b.tok("type"), b.tok(")"))
	plain.Seq(g.Sym("switchStmt"))

	b.caseForms("$commHead", "$comms", "$innerComm", "$lastComm", func(h *parse.Rule) {
		h.Seq(b.tok("case"), expr, b.tok("<-"), expr, b.tok(":"))
		h.Seq(b.tok("case"), list, b.tok("="), expr, b.tok(":"))
		h.Seq(b.tok("case"), list, b.tok(":="), expr, b.tok(":"))
		h.Seq(b.tok("case"), expr, b.tok(":"))
		h.Seq(b.tok("default"), b.tok(":"))
	})
	sel := g.Rule("selectStmt")
	b.braceAlts(sel, "$comms", "$lastComm", b.tok("select"))
	plain.Seq(g.Sym("selectStmt"))

	plain.Seq(g.Sym("constDecl"))
	plain.Seq(g.Sym("varDecl"))
	plain.Seq(g.Sym("typeDecl"))
	plain.Seq(g.Sym("simpleStmt"))
	plain.Seq(b.tok("return"))
	plain.Seq(b.tok("return"), list)
	plain.Seq(b.tok("go"), expr, b.tok("("), g.Sym("args"), b.tok(")"))
	plain.Seq(b.tok("defer"), expr, b.tok("("), g.Sym("args"), b.tok(")"))
	plain.Seq(g.Sym("ifStmt"))
	plain.Seq(g.Sym("forStmt"))
	plain.Seq(b.tok("fallthrough"))
	plain.Seq(b.tok("break"))
	plain.Seq(b.tok("break"), b.tok("IDENT"))
	plain.Seq(b.tok("continue"))
	plain.Seq(b.tok("continue"), b.tok("IDENT"))
	plain.Seq(b.tok("goto"), b.tok("IDENT"))
}

// stmtLists builds statement sequences. A non-label statement carries its
// semicolon. A label does not add a second one. Labels with no statement
// are only legal immediately before '}'.
func (b *bld) stmtLists() {
	g := b.g
	plain := g.Sym("plain")
	term := g.Rule("$term")
	term.Seq(plain, b.tok(";"))
	term.Seq(b.tok(";"))

	lab := g.Rule("$lab")
	lab.Seq(b.tok("IDENT"), b.tok(":"), g.Sym("$labBody"))
	body := g.Rule("$labBody")
	body.Seq(g.Sym("$term"))
	body.Seq(g.Sym("$lab"))
	body.Seq(b.tok(";"))

	seq := g.Rule("$stmtSeq")
	seq.Seq()
	seq.Seq(g.Sym("$stmtSeq"), g.Sym("$term"))
	seq.Seq(g.Sym("$stmtSeq"), g.Sym("$lab"))

	bare := g.Rule("$bareLab")
	bare.Seq(b.tok("IDENT"), b.tok(":"), g.Sym("$bareBody"))
	bb := g.Rule("$bareBody")
	bb.Seq(plain)
	bb.Seq(g.Sym("$bareLab"))

	opt := g.Rule("$optBare")
	opt.Seq()
	opt.Seq(plain)
	opt.Seq(g.Sym("$bareLab"))

	core := g.Rule("$stmtCore")
	core.Seq(g.Sym("$stmtSeq"), g.Sym("$optBare"))

	trail := g.Rule("$trailLab")
	trail.Seq()
	trail.Seq(g.Sym("$trailLab"), b.tok("IDENT"), b.tok(":"))

	list := g.Rule("$stmtList")
	list.Seq(g.Sym("$stmtCore"), g.Sym("$trailLab"))
}

// caseForms builds a bracketed clause list. Every clause except the last
// uses $stmtCore, so a label cannot end on the next clause. The last
// clause uses $stmtList, so a label may end on '}'.
func (b *bld) caseForms(head, cases, inner, last string, alts func(h *parse.Rule)) {
	h := b.g.Rule(head)
	alts(h)
	in := b.g.Rule(inner)
	in.Seq(b.g.Sym(head), b.g.Sym("$stmtCore"))
	ls := b.g.Rule(last)
	ls.Seq(b.g.Sym(head), b.g.Sym("$stmtList"))
	cs := b.g.Rule(cases)
	cs.Seq()
	cs.Seq(b.g.Sym(cases), b.g.Sym(inner))
}

func (b *bld) switchCases(sw *parse.Rule, prefix ...*parse.Symbol) {
	b.braceAlts(sw, "$cases", "$lastCase", prefix...)
}

func (b *bld) switchTypes(sw *parse.Rule, prefix ...*parse.Symbol) {
	b.braceAlts(sw, "$typeCases", "$typeLast", prefix...)
}

func (b *bld) braceAlts(sw *parse.Rule, cases, last string, prefix ...*parse.Symbol) {
	open := append(append([]*parse.Symbol{}, prefix...), b.tok("{"), b.g.Sym(cases), b.tok("}"))
	with := append(append([]*parse.Symbol{}, prefix...), b.tok("{"), b.g.Sym(cases), b.g.Sym(last), b.tok("}"))
	sw.Seq(open...)
	sw.Seq(with...)
}

func (b *bld) declarations() {
	g := b.g
	expr := g.Sym("expr")
	names := g.Sep1(b.tok("IDENT"), b.tok(","))
	list := g.Sep1(expr, b.tok(","))

	val := g.Rule("constVal")
	val.Seq(g.Opt(g.Sym("type_syntax")), b.tok("="), list)
	cs := g.Rule("constSpec")
	cs.Seq(names, g.Opt(g.Sym("constVal")))
	cd := g.Rule("constDecl")
	cd.Seq(b.tok("const"), g.Sym("constSpec"))
	cd.Seq(b.tok("const"), b.tok("("), b.semiBody("constSpecs", g.Sym("constSpec")), b.tok(")"))

	vs := g.Rule("varSpec")
	vs.Seq(names, g.Sym("type_syntax"), b.tok("="), list)
	vs.Seq(names, g.Sym("type_syntax"))
	vs.Seq(names, b.tok("="), list)
	vd := g.Rule("varDecl")
	vd.Seq(b.tok("var"), g.Sym("varSpec"))
	vd.Seq(b.tok("var"), b.tok("("), b.semiBody("varSpecs", g.Sym("varSpec")), b.tok(")"))

	ts := g.Rule("typeSpec")
	ts.Seq(b.tok("IDENT"), g.Opt(g.Sym("type_parameters")), g.Sym("type_syntax"))
	ts.Seq(b.tok("IDENT"), g.Opt(g.Sym("type_parameters")), b.tok("="), g.Sym("type_syntax"))
	td := g.Rule("typeDecl")
	td.Seq(b.tok("type"), g.Sym("typeSpec"))
	td.Seq(b.tok("type"), b.tok("("), b.semiBody("typeSpecs", g.Sym("typeSpec")), b.tok(")"))

	fn := g.Rule("funcDecl")
	fn.Seq(b.tok("func"), b.tok("IDENT"), g.Opt(g.Sym("type_parameters")), g.Sym("sig"), g.Opt(g.Sym("block")))
	md := g.Rule("methodDecl")
	md.Seq(b.tok("func"), g.Sym("params"), b.tok("IDENT"), g.Opt(g.Sym("type_parameters")), g.Sym("sig"), g.Opt(g.Sym("block")))

	is := g.Rule("importSpec")
	is.Seq(b.tok("STRING"))
	is.Seq(b.tok("IDENT"), b.tok("STRING"))
	is.Seq(b.tok("."), b.tok("STRING"))
	im := g.Rule("importDecl")
	im.Seq(b.tok("import"), g.Sym("importSpec"))
	im.Seq(b.tok("import"), b.tok("("), b.semiBody("importSpecs", g.Sym("importSpec")), b.tok(")"))
}

func (b *bld) file() {
	g := b.g
	imps := g.Rule("imports")
	imps.Seq()
	imps.Seq(g.Sym("imports"), g.Sym("importDecl"), b.tok(";"))
	decls := g.Rule("decls")
	decls.Seq()
	for _, name := range []string{"constDecl", "varDecl", "typeDecl", "funcDecl", "methodDecl"} {
		decls.Seq(g.Sym("decls"), g.Sym(name), b.tok(";"))
	}
	file := g.Rule("file")
	file.Seq(b.tok("package"), b.tok("IDENT"), b.tok(";"), g.Sym("imports"), g.Sym("decls"))
}
