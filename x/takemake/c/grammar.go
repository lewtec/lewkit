package c

import (
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

func language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		lang, langErr = define()
	})
	return lang, langErr
}

type gram struct {
	syntax xparse.Syntax[Term, Rule]
	t      map[Term]*xparse.Piece[Term, Rule]
}

func (g *gram) tok(kind Term) *xparse.Piece[Term, Rule] {
	if p, ok := g.t[kind]; ok {
		return p
	}
	p := g.syntax.Term(kind, nil)
	g.t[kind] = p
	return p
}

func (g *gram) ref(name Rule) *xparse.Piece[Term, Rule] {
	return g.syntax.Ref(name)
}

func define() (*xparse.Model[Term, Rule], error) {
	g := &gram{t: map[Term]*xparse.Piece[Term, Rule]{}}
	syntax := g.syntax
	var parts []*xparse.Piece[Term, Rule]

	equation := func(name Rule, body *xparse.Piece[Term, Rule]) {
		parts = append(parts, syntax.Eq(name, body))
	}
	add := func(name Rule, alts ...*xparse.Piece[Term, Rule]) {
		equation(name, syntax.Alt(alts...))
	}

	specs := g.specs()
	equation(RuleSpecifiers, specs)
	add(RuleAggregate, g.aggr()...)
	equation(RuleField, g.field())
	add(RuleEnumerator,
		g.tok(TermIdentifier),
		syntax.Seq(g.tok(TermIdentifier), g.tok(TermEqual), g.ref(RuleExpression)),
	)
	add(RuleEnumeratorList,
		g.ref(RuleEnumerator),
		syntax.Seq(g.ref(RuleEnumeratorList), g.tok(TermComma), g.ref(RuleEnumerator)),
		syntax.Seq(g.ref(RuleEnumeratorList), g.tok(TermComma)),
	)
	add(RulePointer,
		syntax.Seq(g.tok(TermStar), g.quals(), g.ref(RulePointer)),
		syntax.Seq(g.tok(TermStar), g.quals()),
	)
	add(RuleSuffix,
		syntax.Seq(g.tok(TermLeftBracket), syntax.Opt(g.ref(RuleExpression)), g.tok(TermRightBracket)),
		syntax.Seq(g.tok(TermLeftParenthesis), syntax.Opt(g.ref(RuleParameters)), g.tok(TermRightParenthesis)),
	)
	suffixes := syntax.Repeat(g.ref(RuleSuffix))
	// A '(' here is a grouping paren, never a function suffix. Abstract
	// arrays that are not grouped are outside this subset.
	add(RuleDirectDeclarator,
		syntax.Seq(g.tok(TermLeftParenthesis), g.ref(RuleDeclarator), g.tok(TermRightParenthesis), suffixes),
		syntax.Seq(g.tok(TermIdentifier), suffixes),
	)
	add(RuleDeclarator,
		syntax.Seq(g.ref(RulePointer), g.ref(RuleDirectDeclarator)),
		g.ref(RuleDirectDeclarator),
		g.ref(RulePointer),
	)
	add(RuleParameterTail,
		syntax.Seq(),
		syntax.Seq(g.tok(TermComma), g.ref(RuleParameter), g.ref(RuleParameterTail)),
		syntax.Seq(g.tok(TermComma), g.tok(TermEllipsis)),
	)
	add(RuleParameter, syntax.Seq(g.ref(RuleSpecifiers), syntax.Opt(g.ref(RuleDeclarator))))
	add(RuleParameters,
		g.tok(TermEllipsis),
		syntax.Seq(g.ref(RuleParameter), g.ref(RuleParameterTail)),
	)
	equation(RuleTypeName, syntax.Seq(g.ref(RuleSpecifiers), syntax.Opt(g.ref(RuleDeclarator))))
	g.expressions(&parts)
	add(RuleText,
		g.tok(TermString),
		syntax.Seq(g.ref(RuleText), g.tok(TermString)),
	)
	add(RulePrimary,
		g.tok(TermIdentifier),
		g.tok(TermNumber),
		g.tok(TermCharacterLiteral),
		g.ref(RuleText),
		syntax.Seq(g.tok(TermLeftParenthesis), g.ref(RuleExpression), g.tok(TermRightParenthesis)),
	)
	add(RuleArguments, syntax.Seq(g.ref(RuleExpression), syntax.Repeat(syntax.Seq(g.tok(TermComma), g.ref(RuleExpression)))))
	add(RulePostfix,
		g.ref(RulePrimary),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermLeftBracket), g.ref(RuleExpression), g.tok(TermRightBracket)),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermLeftParenthesis), syntax.Opt(g.ref(RuleArguments)), g.tok(TermRightParenthesis)),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermDot), g.tok(TermIdentifier)),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermArrow), g.tok(TermIdentifier)),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermIncrement)),
		syntax.Seq(g.ref(RulePostfix), g.tok(TermDecrement)),
	)
	equation(RuleBlock, syntax.Seq(g.tok(TermLeftBrace), syntax.Repeat(g.ref(RuleStatement)), g.tok(TermRightBrace)))
	add(RuleInitializer,
		g.ref(RuleDeclarator),
		syntax.Seq(g.ref(RuleDeclarator), g.tok(TermEqual), g.ref(RuleExpression)),
	)
	initializers := syntax.Seq(g.ref(RuleInitializer), syntax.Repeat(syntax.Seq(g.tok(TermComma), g.ref(RuleInitializer))))
	add(RuleDeclaration,
		syntax.Seq(g.ref(RuleSpecifiers), g.tok(TermSemicolon)),
		syntax.Seq(g.ref(RuleSpecifiers), initializers, g.tok(TermSemicolon)),
		syntax.Seq(g.ref(RuleSpecifiers), g.ref(RuleDeclarator), g.ref(RuleBlock)),
	)
	equation(RuleLocal, syntax.Seq(g.ref(RuleSpecifiers), initializers, g.tok(TermSemicolon)))
	add(RuleForInitializer,
		g.tok(TermSemicolon),
		g.ref(RuleLocal),
		syntax.Seq(g.ref(RuleExpression), g.tok(TermSemicolon)),
	)
	g.statements(&parts)
	add(RuleDefine,
		syntax.Seq(g.tok(TermDefine), g.tok(TermIdentifier), g.tok(TermNewline)),
		syntax.Seq(g.tok(TermDefine), g.tok(TermIdentifier), g.ref(RuleExpression), g.tok(TermNewline)),
	)
	equation(RuleFile, syntax.Repeat(syntax.Alt(
		g.tok(TermInclude),
		g.tok(TermInvalid),
		g.ref(RuleDefine),
		g.ref(RuleDeclaration),
	)))
	parts = append(parts,
		syntax.Scan(scan),
		syntax.Start(RuleFile),
	)
	return syntax.Compile("c", syntax.Compose(parts...))
}

func (g *gram) quals() *xparse.Piece[Term, Rule] {
	return g.syntax.Repeat(g.syntax.Alt(
		g.tok(TermConst),
		g.tok(TermVolatile),
		g.tok(TermRestrict),
	))
}

func (g *gram) specAtom() *xparse.Piece[Term, Rule] {
	words := []Term{
		TermVoid, TermBool, TermAlternateBool, TermChar, TermShort, TermInt, TermLong,
		TermFloat, TermDouble, TermSigned, TermUnsigned, TermTypedef, TermConst,
		TermVolatile, TermStatic, TermExtern, TermInline, TermRestrict, TermAuto,
		TermRegister, TermNoreturn, TermThreadLocal, TermAlignas, TermAtomic, TermGeneric,
		TermTypeName,
	}
	alts := make([]*xparse.Piece[Term, Rule], 0, len(words)+1)
	for _, w := range words {
		alts = append(alts, g.tok(w))
	}
	alts = append(alts, g.ref(RuleAggregate))
	return g.syntax.Alt(alts...)
}

func (g *gram) specs() *xparse.Piece[Term, Rule] {
	return g.syntax.Repeat1(g.specAtom())
}

func (g *gram) aggr() []*xparse.Piece[Term, Rule] {
	syntax := g.syntax
	body := syntax.Repeat(g.ref(RuleField))
	var out []*xparse.Piece[Term, Rule]
	for _, keyword := range []Term{TermStruct, TermUnion} {
		out = append(out,
			syntax.Seq(g.tok(keyword), g.tok(TermIdentifier), g.tok(TermLeftBrace), body, g.tok(TermRightBrace)),
			syntax.Seq(g.tok(keyword), g.tok(TermLeftBrace), body, g.tok(TermRightBrace)),
			syntax.Seq(g.tok(keyword), g.tok(TermIdentifier)),
		)
	}
	enums := g.ref(RuleEnumeratorList)
	out = append(out,
		syntax.Seq(g.tok(TermEnum), g.tok(TermIdentifier), g.tok(TermLeftBrace), enums, g.tok(TermRightBrace)),
		syntax.Seq(g.tok(TermEnum), g.tok(TermLeftBrace), enums, g.tok(TermRightBrace)),
		syntax.Seq(g.tok(TermEnum), g.tok(TermIdentifier), g.tok(TermLeftBrace), g.tok(TermRightBrace)),
		syntax.Seq(g.tok(TermEnum), g.tok(TermLeftBrace), g.tok(TermRightBrace)),
		syntax.Seq(g.tok(TermEnum), g.tok(TermIdentifier)),
	)
	return out
}

func (g *gram) field() *xparse.Piece[Term, Rule] {
	syntax := g.syntax
	bit := syntax.Seq(g.tok(TermColon), g.ref(RuleExpression))
	one := syntax.Seq(g.ref(RuleDeclarator), syntax.Opt(bit))
	return syntax.Alt(
		syntax.Seq(g.ref(RuleSpecifiers), g.tok(TermSemicolon)),
		syntax.Seq(g.ref(RuleSpecifiers), bit, g.tok(TermSemicolon)),
		syntax.Seq(g.ref(RuleSpecifiers), one, syntax.Repeat(syntax.Seq(g.tok(TermComma), one)), g.tok(TermSemicolon)),
	)
}

func (g *gram) expressions(parts *[]*xparse.Piece[Term, Rule]) {
	syntax := g.syntax
	bin := func(op Term, prec int) {
		*parts = append(*parts, syntax.Eq(RuleExpression, syntax.Left(prec, syntax.Seq(g.ref(RuleExpression), g.tok(op), g.ref(RuleExpression)))))
	}
	asg := func(op Term) {
		*parts = append(*parts, syntax.Eq(RuleExpression, syntax.Right(1, syntax.Seq(g.ref(RuleExpression), g.tok(op), g.ref(RuleExpression)))))
	}
	asg(TermEqual)
	for _, op := range []Term{TermAddEqual, TermSubtractEqual, TermMultiplyEqual, TermDivideEqual, TermModuloEqual, TermAndEqual, TermOrEqual, TermXorEqual, TermShiftLeftEqual, TermShiftRightEqual} {
		asg(op)
	}
	bin(TermLogicalOr, 2)
	bin(TermLogicalAnd, 3)
	bin(TermBitwiseOr, 4)
	bin(TermBitwiseXor, 5)
	bin(TermBitwiseAnd, 6)
	bin(TermEqualEqual, 7)
	bin(TermNotEqual, 7)
	bin(TermLess, 8)
	bin(TermGreater, 8)
	bin(TermLessEqual, 8)
	bin(TermGreaterEqual, 8)
	bin(TermShiftLeft, 9)
	bin(TermShiftRight, 9)
	bin(TermPlus, 10)
	bin(TermMinus, 10)
	bin(TermStar, 11)
	bin(TermSlash, 11)
	bin(TermPercent, 11)
	*parts = append(*parts, syntax.Eq(RuleExpression, g.ref(RuleUnary)))
	// Cast stays outside sizeof. sizeof (type) and sizeof (cast) would
	// otherwise both cover sizeof(T)+x.
	*parts = append(*parts, syntax.Eq(RuleUnary, syntax.Seq(g.tok(TermLeftParenthesis), g.ref(RuleTypeName), g.tok(TermRightParenthesis), g.ref(RuleUnary))))
	*parts = append(*parts, syntax.Eq(RuleUnary, g.ref(RulePrefix)))

	prefix := func(body *xparse.Piece[Term, Rule]) {
		*parts = append(*parts, syntax.Eq(RulePrefix, body))
	}
	for _, op := range []Term{TermStar, TermBitwiseAnd, TermLogicalNot, TermBitwiseNot, TermMinus, TermPlus, TermIncrement, TermDecrement} {
		prefix(syntax.Seq(g.tok(op), g.ref(RulePrefix)))
	}
	prefix(syntax.Seq(g.tok(TermSizeof), g.ref(RulePrefix)))
	prefix(syntax.Seq(g.tok(TermSizeof), g.tok(TermLeftParenthesis), g.ref(RuleTypeName), g.tok(TermRightParenthesis)))
	prefix(g.ref(RulePostfix))
}

func (g *gram) statements(parts *[]*xparse.Piece[Term, Rule]) {
	syntax := g.syntax
	statement := func(prec int, body *xparse.Piece[Term, Rule]) {
		if prec > 0 {
			body = syntax.Prec(prec, body)
		}
		*parts = append(*parts, syntax.Eq(RuleStatement, body))
	}
	condition := syntax.Seq(g.tok(TermLeftParenthesis), g.ref(RuleExpression), g.tok(TermRightParenthesis))
	statement(1, syntax.Seq(g.tok(TermIf), condition, g.ref(RuleStatement)))
	statement(2, syntax.Seq(g.tok(TermIf), condition, g.ref(RuleStatement), g.tok(TermElse), g.ref(RuleStatement)))
	statement(0, syntax.Seq(g.tok(TermWhile), condition, g.ref(RuleStatement)))
	statement(0, syntax.Seq(g.tok(TermFor), g.tok(TermLeftParenthesis), g.ref(RuleForInitializer), syntax.Opt(g.ref(RuleExpression)), g.tok(TermSemicolon), syntax.Opt(g.ref(RuleExpression)), g.tok(TermRightParenthesis), g.ref(RuleStatement)))
	statement(0, syntax.Seq(g.tok(TermReturn), syntax.Opt(g.ref(RuleExpression)), g.tok(TermSemicolon)))
	statement(0, syntax.Seq(g.tok(TermBreak), g.tok(TermSemicolon)))
	statement(0, syntax.Seq(g.tok(TermContinue), g.tok(TermSemicolon)))
	statement(0, syntax.Seq(g.tok(TermSwitch), condition, g.ref(RuleStatement)))
	statement(0, syntax.Seq(g.tok(TermDo), g.ref(RuleStatement), g.tok(TermWhile), condition, g.tok(TermSemicolon)))
	statement(0, syntax.Seq(g.tok(TermGoto), g.tok(TermIdentifier), g.tok(TermSemicolon)))
	statement(0, g.ref(RuleBlock))
	statement(0, g.ref(RuleLocal))
	statement(0, syntax.Seq(g.ref(RuleExpression), g.tok(TermSemicolon)))
}
