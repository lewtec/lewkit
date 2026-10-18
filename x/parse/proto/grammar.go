package proto

import (
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

// Language loads the proto grammar.
func Language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		prepared, err := declare()
		if err != nil {
			langErr = err
			return
		}
		lang, langErr = prepared.Load(protoTable)
	})
	return lang, langErr
}

func declare() (*xparse.Prepared[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]
	identifier := syntax.Word(TermIdentifier, identifierStart, identifierContinue)
	stringLiteral := syntax.Pat(TermString, xparse.Func(matchString))
	integerLiteral := syntax.Pat(TermInteger, xparse.Func(matchInteger))
	floatLiteral := syntax.Pat(TermFloat, xparse.Func(matchFloat))
	// Keyword shares the ASCII identifier boundary. Pat(Keyword) is that boundary.
	key := func(kind Term) *xparse.Piece[Term, Rule] {
		return syntax.Pat(kind, xparse.Keyword(string(kind)))
	}
	kwSyntax := key(TermSyntax)
	kwPackage := key(TermPackage)
	kwImport := key(TermImport)
	kwPublic := key(TermPublic)
	kwWeak := key(TermWeak)
	kwOption := key(TermOption)
	kwMessage := key(TermMessage)
	kwEnum := key(TermEnum)
	kwService := key(TermService)
	kwReserved := key(TermReserved)
	kwOneof := key(TermOneof)
	kwRPC := key(TermRPC)
	kwReturns := key(TermReturns)
	kwStream := key(TermStream)
	kwOptional := key(TermOptional)
	kwRequired := key(TermRequired)
	kwRepeated := key(TermRepeated)
	kwExtend := key(TermExtend)
	kwExtensions := key(TermExtensions)
	kwGroup := key(TermGroup)
	kwTo := key(TermTo)
	kwMax := key(TermMax)

	semicolon := syntax.Lit(";")
	equal := syntax.Lit("=")
	colon := syntax.Lit(":")
	leftBrace := syntax.Lit("{")
	rightBrace := syntax.Lit("}")
	leftParenthesis := syntax.Lit("(")
	rightParenthesis := syntax.Lit(")")
	leftBracket := syntax.Lit("[")
	rightBracket := syntax.Lit("]")
	less := syntax.Lit("<")
	greater := syntax.Lit(">")
	comma := syntax.Lit(",")
	dot := syntax.Lit(".")
	plus := syntax.Lit("+")
	minus := syntax.Lit("-")

	fullIdentifier := syntax.Eq(RuleFullIdentifier, syntax.Seq(identifier, syntax.Repeat(syntax.Seq(dot, identifier))))
	typeName := syntax.Eq(RuleTypeName, syntax.Seq(syntax.Opt(dot), syntax.Ref(RuleFullIdentifier)))
	// map is an identifier before "<", not a keyword. A keyword would sit in
	// the same state as a type name and steal a field whose type is named map.
	typeExpression := syntax.Eq(RuleType, syntax.Alt(
		syntax.Seq(identifier, less, syntax.Ref(RuleType), comma, syntax.Ref(RuleType), greater),
		syntax.Ref(RuleTypeName),
	))
	strings := syntax.Eq(RuleStrings, syntax.Repeat1(stringLiteral))
	scalar := syntax.Eq(RuleScalar, syntax.Alt(
		syntax.Ref(RuleStrings),
		integerLiteral,
		floatLiteral,
		syntax.Ref(RuleFullIdentifier),
	))
	optionName := syntax.Eq(RuleOptionName, syntax.Alt(
		syntax.Seq(leftParenthesis, syntax.Ref(RuleFullIdentifier), rightParenthesis, syntax.Repeat(syntax.Seq(dot, identifier))),
		syntax.Ref(RuleFullIdentifier),
	))
	constant := syntax.Eq(RuleConstant, syntax.Alt(
		syntax.Seq(leftBrace, syntax.Repeat(syntax.Alt(comma, semicolon, syntax.Ref(RuleAssign))), rightBrace),
		syntax.Seq(leftBracket, syntax.Opt(syntax.Sep1(syntax.Ref(RuleConstant), comma)), rightBracket),
		syntax.Seq(syntax.Opt(syntax.Alt(plus, minus)), syntax.Ref(RuleScalar)),
	))
	assign := syntax.Eq(RuleAssign, syntax.Seq(
		syntax.Ref(RuleOptionName),
		syntax.Alt(equal, colon),
		syntax.Ref(RuleConstant),
	))
	brackets := syntax.Eq(RuleBrackets, syntax.Seq(
		leftBracket,
		syntax.Opt(syntax.Sep1(syntax.Ref(RuleAssign), comma)),
		rightBracket,
	))
	span := syntax.Eq(RuleRange, syntax.Seq(
		integerLiteral,
		syntax.Opt(syntax.Seq(kwTo, syntax.Alt(kwMax, integerLiteral))),
	))
	reserved := syntax.Eq(RuleReserved, syntax.Seq(
		kwReserved,
		syntax.Alt(
			syntax.Sep1(syntax.Ref(RuleStrings), comma),
			syntax.Sep1(syntax.Ref(RuleRange), comma),
		),
		semicolon,
	))
	field := syntax.Eq(RuleField, syntax.Seq(
		syntax.Opt(syntax.Alt(kwOptional, kwRequired, kwRepeated)),
		syntax.Ref(RuleType),
		identifier,
		equal,
		integerLiteral,
		syntax.Opt(syntax.Ref(RuleBrackets)),
		semicolon,
	))
	enumValue := syntax.Eq(RuleEnumValue, syntax.Seq(
		identifier,
		equal,
		syntax.Opt(syntax.Alt(plus, minus)),
		integerLiteral,
		syntax.Opt(syntax.Ref(RuleBrackets)),
		semicolon,
	))
	option := syntax.Eq(RuleOption, syntax.Seq(kwOption, syntax.Ref(RuleAssign), semicolon))
	rpcType := syntax.Eq(RuleRPCType, syntax.Seq(syntax.Opt(kwStream), syntax.Ref(RuleTypeName)))
	rpc := syntax.Eq(RuleRPC, syntax.Seq(
		kwRPC,
		identifier,
		leftParenthesis,
		syntax.Ref(RuleRPCType),
		rightParenthesis,
		kwReturns,
		leftParenthesis,
		syntax.Ref(RuleRPCType),
		rightParenthesis,
		syntax.Alt(
			semicolon,
			syntax.Seq(leftBrace, syntax.Repeat(syntax.Alt(semicolon, syntax.Ref(RuleOption))), rightBrace),
		),
	))
	// Enum options and reserves start with an identifier, not the keyword.
	// The keyword would share this state with an enumerator name and steal
	// "option = 1" and "reserved = 1". The next token separates them.
	enumOption := syntax.Eq(RuleEnumOption, syntax.Seq(identifier, syntax.Ref(RuleAssign), semicolon))
	enumReserved := syntax.Eq(RuleEnumReserved, syntax.Seq(
		identifier,
		syntax.Alt(
			syntax.Sep1(syntax.Ref(RuleStrings), comma),
			syntax.Sep1(syntax.Ref(RuleRange), comma),
		),
		semicolon,
	))
	enum := syntax.Eq(RuleEnum, syntax.Seq(
		kwEnum,
		identifier,
		leftBrace,
		syntax.Repeat(syntax.Alt(semicolon, syntax.Ref(RuleEnumOption), syntax.Ref(RuleEnumReserved), syntax.Ref(RuleEnumValue))),
		rightBrace,
	))
	oneof := syntax.Eq(RuleOneof, syntax.Seq(
		kwOneof,
		identifier,
		leftBrace,
		syntax.Repeat(syntax.Alt(semicolon, syntax.Ref(RuleOption), syntax.Ref(RuleField))),
		rightBrace,
	))
	message := syntax.Eq(RuleMessage, syntax.Seq(
		kwMessage,
		identifier,
		leftBrace,
		syntax.Repeat(syntax.Alt(
			semicolon,
			kwExtend,
			kwExtensions,
			kwGroup,
			syntax.Ref(RuleOption),
			syntax.Ref(RuleReserved),
			syntax.Ref(RuleEnum),
			syntax.Ref(RuleMessage),
			syntax.Ref(RuleOneof),
			syntax.Ref(RuleField),
		)),
		rightBrace,
	))
	service := syntax.Eq(RuleService, syntax.Seq(
		kwService,
		identifier,
		leftBrace,
		syntax.Repeat(syntax.Alt(semicolon, syntax.Ref(RuleOption), syntax.Ref(RuleRPC))),
		rightBrace,
	))
	syntaxDeclaration := syntax.Eq(RuleSyntax, syntax.Seq(kwSyntax, equal, syntax.Ref(RuleStrings), semicolon))
	packageDeclaration := syntax.Eq(RulePackage, syntax.Seq(kwPackage, syntax.Ref(RuleFullIdentifier), semicolon))
	importDeclaration := syntax.Eq(RuleImport, syntax.Seq(
		kwImport,
		syntax.Opt(syntax.Alt(kwPublic, kwWeak)),
		syntax.Ref(RuleStrings),
		semicolon,
	))
	file := syntax.Eq(RuleFile, syntax.Repeat(syntax.Alt(
		semicolon,
		syntax.Ref(RuleSyntax),
		syntax.Ref(RulePackage),
		syntax.Ref(RuleImport),
		syntax.Ref(RuleOption),
		syntax.Ref(RuleMessage),
		syntax.Ref(RuleEnum),
		syntax.Ref(RuleService),
	)))
	return syntax.Prepare("proto", syntax.Compose(
		syntax.Skip(syntax.Pat(TermSpace, xparse.While(" \t\r\v\f\n"))),
		syntax.Skip(syntax.Pat(TermComment, xparse.Func(matchComment))),
		file, syntaxDeclaration, packageDeclaration, importDeclaration, option, message, enum, service, oneof, field,
		enumValue, enumOption, enumReserved, rpc, rpcType, reserved, span, typeExpression, typeName, fullIdentifier,
		constant, scalar, strings, assign, optionName, brackets,
		syntax.Start(RuleFile),
	))
}
