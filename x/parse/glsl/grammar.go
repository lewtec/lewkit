package glsl

import (
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

// Precedence numbers are stored in the generated table.
const (
	precComma = iota + 1
	precAssign
	precTernary
	precOr
	precXor
	precAnd
	precBitOr
	precBitXor
	precBitAnd
	precEq
	precRel
	precShift
	precAdd
	precMul
	precUnary
	precPost
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

// Language loads the GLSL grammar.
func Language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		prepared, err := declare()
		if err != nil {
			langErr = err
			return
		}
		lang, langErr = prepared.Load(glslTable)
	})
	return lang, langErr
}

type gram struct {
	syntax     xparse.Syntax[Term, Rule]
	identifier *xparse.Piece[Term, Rule]
	operators  map[Term]*xparse.Piece[Term, Rule]
	keywords   map[Term]*xparse.Piece[Term, Rule]
}

func (g *gram) keyword(kind Term) *xparse.Piece[Term, Rule] {
	if p, ok := g.keywords[kind]; ok {
		return p
	}
	p := g.syntax.Pat(kind, xparse.Keyword(string(kind)))
	if g.keywords == nil {
		g.keywords = map[Term]*xparse.Piece[Term, Rule]{}
	}
	g.keywords[kind] = p
	return p
}

func (g *gram) op(kind Term) *xparse.Piece[Term, Rule] {
	if p, ok := g.operators[kind]; ok {
		return p
	}
	p := g.syntax.Pat(kind, xparse.Literal(string(kind)))
	if g.operators == nil {
		g.operators = map[Term]*xparse.Piece[Term, Rule]{}
	}
	g.operators[kind] = p
	return p
}

func (g gram) ref(name Rule) *xparse.Piece[Term, Rule] {
	return g.syntax.Ref(name)
}

func (g gram) seq(parts ...*xparse.Piece[Term, Rule]) *xparse.Piece[Term, Rule] {
	return g.syntax.Seq(parts...)
}

func declare() (*xparse.Prepared[Term, Rule], error) {
	var g gram
	g.identifier = g.syntax.Word(TermIdentifier, identifierStart, identifierContinue)
	space := g.syntax.Pat(TermSpace, xparse.While(" \t\n\v\f\r"))

	leftParenthesis := g.op(TermLeftParenthesis)
	rightParenthesis := g.op(TermRightParenthesis)
	leftBracket := g.op(TermLeftBracket)
	rightBracket := g.op(TermRightBracket)
	leftBrace := g.op(TermLeftBrace)
	rightBrace := g.op(TermRightBrace)
	dot := g.op(TermDot)
	comma := g.op(TermComma)
	semicolon := g.op(TermSemicolon)
	colon := g.op(TermColon)
	question := g.op(TermQuestion)
	equal := g.op(TermEqual)

	number := g.syntax.Pat(TermNumber, xparse.Func(matchNumber))
	quals := g.syntax.Repeat(g.ref(RuleQualifier))
	brackets := g.syntax.Repeat(g.ref(RuleBracket))
	expression := g.ref(RuleExpression)
	assign := g.ref(RuleAssign)
	statement := g.ref(RuleStatement)
	declarators := g.ref(RuleDeclarators)

	bin := []struct {
		op   Term
		prec int
	}{
		{TermLogicalOr, precOr},
		{TermLogicalXor, precXor},
		{TermLogicalAnd, precAnd},
		{TermPipe, precBitOr},
		{TermCaret, precBitXor},
		{TermAmpersand, precBitAnd},
		{TermEqualEqual, precEq},
		{TermExclamationEqual, precEq},
		{TermLess, precRel},
		{TermGreater, precRel},
		{TermLessEqual, precRel},
		{TermGreaterEqual, precRel},
		{TermShiftLeft, precShift},
		{TermShiftRight, precShift},
		{TermPlus, precAdd},
		{TermMinus, precAdd},
		{TermStar, precMul},
		{TermSlash, precMul},
		{TermPercent, precMul},
	}
	assignOp := []Term{
		TermEqual, TermPlusEqual, TermMinusEqual, TermStarEqual, TermSlashEqual, TermPercentEqual,
		TermShiftLeftEqual, TermShiftRightEqual, TermAmpersandEqual, TermCaretEqual, TermPipeEqual,
	}
	prefix := []Term{TermIncrement, TermDecrement, TermPlus, TermMinus, TermExclamation, TermTilde}

	assignAlts := []*xparse.Piece[Term, Rule]{
		g.syntax.Right(precTernary, g.seq(assign, question, assign, colon, assign)),
		g.syntax.Left(precPost, g.seq(assign, leftBracket, assign, rightBracket)),
		g.syntax.Left(precPost, g.seq(assign, dot, g.identifier)),
		g.syntax.Left(precPost, g.seq(assign, leftParenthesis, g.syntax.Opt(g.ref(RuleArguments)), rightParenthesis)),
		g.syntax.Left(precPost, g.seq(assign, g.op(TermIncrement))),
		g.syntax.Left(precPost, g.seq(assign, g.op(TermDecrement))),
		g.identifier,
		number,
		g.seq(leftParenthesis, expression, rightParenthesis),
	}
	for _, item := range bin {
		assignAlts = append(assignAlts, g.syntax.Left(item.prec, g.seq(assign, g.op(item.op), assign)))
	}
	for _, op := range assignOp {
		assignAlts = append(assignAlts, g.syntax.Right(precAssign, g.seq(assign, g.op(op), assign)))
	}
	for _, op := range prefix {
		assignAlts = append(assignAlts, g.syntax.Right(precUnary, g.seq(g.op(op), assign)))
	}

	parts := []*xparse.Piece[Term, Rule]{
		g.syntax.Skip(space),
		g.syntax.Start(RuleFile),

		g.syntax.Eq(RuleFile, g.syntax.Repeat(g.syntax.Alt(
			g.ref(RulePrecisionStatement),
			g.ref(RuleStructDeclaration),
			g.ref(RuleInterface),
			g.ref(RuleFunc),
			g.ref(RuleDeclaration),
			g.ref(RuleNamed),
			g.ref(RuleBare),
		))),

		g.syntax.Eq(RulePrecisionStatement, g.seq(g.keyword(TermPrecision), g.ref(RuleRank), g.identifier, semicolon)),
		g.syntax.Eq(RuleRank, g.syntax.Alt(g.keyword(TermLowp), g.keyword(TermMediump), g.keyword(TermHighp))),

		g.syntax.Eq(RuleStructDeclaration, g.seq(
			g.keyword(TermStruct), g.identifier, leftBrace, g.syntax.Repeat(g.ref(RuleDeclaration)), rightBrace, g.ref(RuleTail),
		)),
		g.syntax.Eq(RuleTail, g.syntax.Alt(
			semicolon,
			g.seq(declarators, semicolon),
		)),

		g.syntax.Eq(RuleDeclaration, g.seq(quals, g.identifier, declarators, semicolon)),
		g.syntax.Eq(RuleDeclarators, g.syntax.Sep1(g.ref(RuleDeclarator), comma)),
		g.syntax.Eq(RuleDeclarator, g.seq(g.identifier, brackets, g.syntax.Opt(g.ref(RuleInitializer)))),
		g.syntax.Eq(RuleBracket, g.syntax.Alt(
			g.seq(leftBracket, rightBracket),
			g.seq(leftBracket, assign, rightBracket),
		)),
		g.syntax.Eq(RuleInitializer, g.syntax.Alt(
			g.seq(equal, assign),
			g.seq(equal, g.ref(RuleList)),
		)),
		g.syntax.Eq(RuleList, g.syntax.Alt(
			g.seq(leftBrace, rightBrace),
			g.seq(leftBrace, g.ref(RuleElements), rightBrace),
			g.seq(leftBrace, g.ref(RuleElements), comma, rightBrace),
		)),
		g.syntax.Eq(RuleElements, g.syntax.Alt(
			g.ref(RuleElement),
			g.seq(g.ref(RuleElements), comma, g.ref(RuleElement)),
		)),
		g.syntax.Eq(RuleElement, g.syntax.Alt(assign, g.ref(RuleList))),

		g.syntax.Eq(RuleInterface, g.seq(
			quals, g.identifier, leftBrace, g.syntax.Repeat(g.ref(RuleDeclaration)), rightBrace,
			g.syntax.Opt(g.ref(RuleInstance)), semicolon,
		)),
		g.syntax.Eq(RuleInstance, g.seq(g.identifier, brackets)),
		// A storage qualifier and a bare name, as in "uniform color;".
		// Repeat1 keeps a lone identifier for the declaration rule.
		g.syntax.Eq(RuleNamed, g.seq(g.syntax.Repeat1(g.ref(RuleQualifier)), g.identifier, semicolon)),
		g.syntax.Eq(RuleBare, g.seq(quals, semicolon)),

		g.syntax.Eq(RuleQualifier, g.syntax.Alt(
			g.ref(RuleLayoutQualifier),
			g.keyword(TermInvariant), g.keyword(TermPrecise),
			g.keyword(TermSmooth), g.keyword(TermFlat), g.keyword(TermNoPerspective),
			g.keyword(TermCentroid), g.keyword(TermSample), g.keyword(TermPatch),
			g.keyword(TermCoherent), g.keyword(TermVolatile), g.keyword(TermRestrict), g.keyword(TermReadonly), g.keyword(TermWriteonly),
			g.keyword(TermLowp), g.keyword(TermMediump), g.keyword(TermHighp),
			g.keyword(TermConst), g.keyword(TermIn), g.keyword(TermOut), g.keyword(TermInOut),
			g.keyword(TermUniform), g.keyword(TermBuffer), g.keyword(TermShared),
		)),
		g.syntax.Eq(RuleLayoutQualifier, g.seq(g.keyword(TermLayout), leftParenthesis, g.syntax.Opt(g.ref(RuleLayoutItems)), rightParenthesis)),
		g.syntax.Eq(RuleLayoutItems, g.syntax.Sep1(g.seq(g.identifier, g.syntax.Opt(g.seq(equal, assign))), comma)),

		g.syntax.Eq(RuleFunc, g.seq(
			quals, g.identifier, g.identifier, leftParenthesis, g.syntax.Opt(g.ref(RuleParameterList)), rightParenthesis,
			brackets, g.syntax.Alt(semicolon, g.ref(RuleBody)),
		)),
		g.syntax.Eq(RuleParameterList, g.syntax.Sep1(g.ref(RuleParameter), comma)),
		g.syntax.Eq(RuleParameter, g.seq(
			g.syntax.Repeat(g.ref(RuleParameterQualifier)), g.identifier, g.ref(RuleParameterName),
		)),
		// Brackets before the name are an unnamed array parameter.
		// Brackets after the name belong to the named parameter.
		// The two shapes do not share a nullable suffix, so "[" is not a fork.
		g.syntax.Eq(RuleParameterName, g.syntax.Alt(
			brackets,
			g.seq(g.identifier, brackets),
		)),
		g.syntax.Eq(RuleParameterQualifier, g.syntax.Alt(
			g.keyword(TermConst), g.keyword(TermIn), g.keyword(TermOut), g.keyword(TermInOut),
			g.keyword(TermLowp), g.keyword(TermMediump), g.keyword(TermHighp),
			g.keyword(TermCoherent), g.keyword(TermVolatile), g.keyword(TermRestrict), g.keyword(TermReadonly), g.keyword(TermWriteonly),
		)),
		g.syntax.Eq(RuleBody, g.seq(leftBrace, g.syntax.Repeat(statement), rightBrace)),

		g.syntax.Eq(RuleStatement, g.syntax.Alt(
			g.ref(RuleEmpty),
			g.ref(RuleBlock),
			g.ref(RuleIfStatement),
			g.ref(RuleForStatement),
			g.ref(RuleWhileStatement),
			g.ref(RuleDoStatement),
			g.ref(RuleSwitchStatement),
			g.ref(RuleReturnStatement),
			g.ref(RuleBreakStatement),
			g.ref(RuleContinueStatement),
			g.ref(RuleDiscardStatement),
			g.ref(RuleDeclaration),
			g.ref(RuleExpressionStatement),
		)),
		g.syntax.Eq(RuleEmpty, semicolon),
		g.syntax.Eq(RuleBlock, g.seq(leftBrace, g.syntax.Repeat(statement), rightBrace)),
		// The else alternative shifts. The bare if reduces at a lower precedence.
		g.syntax.Eq(RuleIfStatement, g.syntax.Alt(
			g.syntax.Prec(1, g.seq(g.keyword(TermIf), leftParenthesis, expression, rightParenthesis, statement)),
			g.syntax.Prec(2, g.seq(g.keyword(TermIf), leftParenthesis, expression, rightParenthesis, statement, g.keyword(TermElse), statement)),
		)),
		g.syntax.Eq(RuleForStatement, g.syntax.Alt(
			g.seq(g.keyword(TermFor), leftParenthesis, semicolon, g.syntax.Opt(expression), semicolon, g.syntax.Opt(expression), rightParenthesis, statement),
			g.seq(g.keyword(TermFor), leftParenthesis, expression, semicolon, g.syntax.Opt(expression), semicolon, g.syntax.Opt(expression), rightParenthesis, statement),
			g.seq(g.keyword(TermFor), leftParenthesis, g.ref(RuleDeclaration), g.syntax.Opt(expression), semicolon, g.syntax.Opt(expression), rightParenthesis, statement),
		)),
		g.syntax.Eq(RuleWhileStatement, g.seq(g.keyword(TermWhile), leftParenthesis, expression, rightParenthesis, statement)),
		g.syntax.Eq(RuleDoStatement, g.seq(g.keyword(TermDo), statement, g.keyword(TermWhile), leftParenthesis, expression, rightParenthesis, semicolon)),
		g.syntax.Eq(RuleSwitchStatement, g.seq(
			g.keyword(TermSwitch), leftParenthesis, expression, rightParenthesis, leftBrace, g.syntax.Repeat(g.ref(RuleSwitchItem)), rightBrace,
		)),
		g.syntax.Eq(RuleSwitchItem, g.syntax.Alt(g.ref(RuleCaseLabel), g.ref(RuleDefaultLabel), statement)),
		g.syntax.Eq(RuleCaseLabel, g.seq(g.keyword(TermCase), assign, colon)),
		g.syntax.Eq(RuleDefaultLabel, g.seq(g.keyword(TermDefault), colon)),
		g.syntax.Eq(RuleReturnStatement, g.syntax.Alt(
			g.seq(g.keyword(TermReturn), semicolon),
			g.seq(g.keyword(TermReturn), expression, semicolon),
		)),
		g.syntax.Eq(RuleBreakStatement, g.seq(g.keyword(TermBreak), semicolon)),
		g.syntax.Eq(RuleContinueStatement, g.seq(g.keyword(TermContinue), semicolon)),
		g.syntax.Eq(RuleDiscardStatement, g.seq(g.keyword(TermDiscard), semicolon)),
		g.syntax.Eq(RuleExpressionStatement, g.seq(expression, semicolon)),

		g.syntax.Eq(RuleExpression, g.syntax.Alt(
			assign,
			g.syntax.Left(precComma, g.seq(expression, comma, assign)),
		)),
		g.syntax.Eq(RuleAssign, g.syntax.Alt(assignAlts...)),
		g.syntax.Eq(RuleArguments, g.syntax.Sep1(assign, comma)),
	}
	return g.syntax.Prepare("glsl", g.syntax.Compose(parts...))
}

func identifierStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func identifierContinue(r rune) bool {
	return identifierStart(r) || r >= '0' && r <= '9'
}

func matchNumber(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	if src[at] != '.' && !isDigit(src[at]) {
		return 0
	}
	if src[at] == '.' && (at+1 >= len(src) || !isDigit(src[at+1])) {
		return 0
	}
	_, end, err := ScanNumber(string(src), at, 1)
	if err != nil || end <= at {
		return 0
	}
	return end - at
}
