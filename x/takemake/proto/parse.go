package proto

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

const maxFieldNumber int64 = 536870911

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

// Take reads proto text into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader", ErrValue)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValue, err)
	}
	if len(body) >= 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		body = body[3:]
	}
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse(body)
	if err != nil {
		var pe *xparse.Error
		if errors.As(err, &pe) {
			return nil, syntaxError(pe.Line, "syntax")
		}
		return nil, syntaxError(1, "syntax")
	}
	return lower(string(body), root)
}

func define() (*xparse.Model[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]
	identifier := syntax.Word(TermIdentifier, identifierStart, identifierContinue)
	stringLiteral := syntax.Term(TermString, matchString)
	integerLiteral := syntax.Term(TermInteger, matchInteger)
	floatLiteral := syntax.Term(TermFloat, matchFloat)
	key := func(kind Term) *xparse.Piece[Term, Rule] {
		return syntax.Key(kind, TermIdentifier)
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
	return syntax.Compile("proto", syntax.Compose(
		syntax.Skip(syntax.Term(TermSpace, matchSpace)),
		syntax.Skip(syntax.Term(TermComment, matchComment)),
		file, syntaxDeclaration, packageDeclaration, importDeclaration, option, message, enum, service, oneof, field,
		enumValue, enumOption, enumReserved, rpc, rpcType, reserved, span, typeExpression, typeName, fullIdentifier,
		constant, scalar, strings, assign, optionName, brackets,
		syntax.Start(RuleFile),
	))
}

func lower(src string, root xparse.Syn[Term, Rule]) (*File, error) {
	if rule, ok := root.Rule(); !ok || rule != RuleFile {
		return nil, syntaxAt(src, root, "syntax")
	}
	file := NewFile()
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleSyntax:
			if file.Syntax != "" {
				return nil, syntaxAt(src, child, "syntax")
			}
			text, err := declStrings(src, child)
			if err != nil {
				return nil, err
			}
			file.Syntax = text
		case RulePackage:
			if file.Package != "" {
				return nil, syntaxAt(src, child, "package")
			}
			file.Package = declName(child)
		case RuleImport:
			imported, err := lowerImport(src, child)
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, imported)
		case RuleOption:
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			file.Options = append(file.Options, option)
		case RuleMessage:
			message, err := lowerMessage(src, child)
			if err != nil {
				return nil, err
			}
			file.Messages = append(file.Messages, message)
		case RuleEnum:
			enum, err := lowerEnum(src, child)
			if err != nil {
				return nil, err
			}
			file.Enums = append(file.Enums, enum)
		case RuleService:
			service, err := lowerService(src, child)
			if err != nil {
				return nil, err
			}
			file.Services = append(file.Services, service)
		}
	}
	return file, nil
}

func lowerImport(src string, node xparse.Syn[Term, Rule]) (Import, error) {
	imported := Import{}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok {
			switch term {
			case TermPublic:
				imported.Kind = ImportPublic
			case TermWeak:
				imported.Kind = ImportWeak
			}
			continue
		}
		if rule, ok := child.Rule(); ok && rule == RuleStrings {
			text, err := lowerStrings(src, child)
			if err != nil {
				return Import{}, err
			}
			imported.Path = text
		}
	}
	return imported, nil
}

func lowerOption(src string, node xparse.Syn[Term, Rule]) (Option, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleAssign {
			return lowerAssign(src, child)
		}
	}
	return Option{}, syntaxAt(src, node, "expected option")
}

func lowerMessage(src string, node xparse.Syn[Term, Rule]) (*Message, error) {
	message := &Message{Name: declName(node)}
	if message.Name == "" {
		return nil, syntaxAt(src, node, "expected identifier")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok {
			switch term {
			case TermExtend, TermExtensions, TermGroup:
				return nil, syntaxAt(src, child, child.Text())
			}
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleOption:
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			message.Options = append(message.Options, option)
		case RuleReserved:
			reserved, err := lowerReserved(src, child)
			if err != nil {
				return nil, err
			}
			message.Reserved = append(message.Reserved, reserved)
		case RuleEnum:
			enum, err := lowerEnum(src, child)
			if err != nil {
				return nil, err
			}
			message.Enums = append(message.Enums, enum)
		case RuleMessage:
			nested, err := lowerMessage(src, child)
			if err != nil {
				return nil, err
			}
			message.Messages = append(message.Messages, nested)
		case RuleOneof:
			oneof, err := lowerOneof(src, child)
			if err != nil {
				return nil, err
			}
			message.Oneofs = append(message.Oneofs, oneof)
		case RuleField:
			field, err := lowerField(src, child)
			if err != nil {
				return nil, err
			}
			message.Fields = append(message.Fields, field)
		}
	}
	return message, nil
}

func lowerOneof(src string, node xparse.Syn[Term, Rule]) (*Oneof, error) {
	oneof := &Oneof{Name: declName(node)}
	if oneof.Name == "" {
		return nil, syntaxAt(src, node, "expected identifier")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleOption:
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			oneof.Options = append(oneof.Options, option)
		case RuleField:
			field, err := lowerField(src, child)
			if err != nil {
				return nil, err
			}
			oneof.Fields = append(oneof.Fields, field)
		}
	}
	return oneof, nil
}

func lowerField(src string, node xparse.Syn[Term, Rule]) (*Field, error) {
	field := &Field{}
	sawNumber := false
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok {
			switch term {
			case TermOptional:
				field.Label = LabelOptional
			case TermRequired:
				field.Label = LabelRequired
			case TermRepeated:
				field.Label = LabelRepeated
			case TermIdentifier:
				field.Name = child.Text()
			case TermInteger:
				number, err := fieldNumber(src, child)
				if err != nil {
					return nil, err
				}
				field.Number = number
				sawNumber = true
			}
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleType:
			typ, err := lowerType(src, child)
			if err != nil {
				return nil, err
			}
			if typ.Name == "group" {
				return nil, syntaxAt(src, child, "group")
			}
			field.Type = typ
		case RuleBrackets:
			options, err := lowerBrackets(src, child)
			if err != nil {
				return nil, err
			}
			field.Options = options
		}
	}
	if field.Name == "" || !sawNumber {
		return nil, syntaxAt(src, node, "expected field")
	}
	return field, nil
}

func lowerEnum(src string, node xparse.Syn[Term, Rule]) (*Enum, error) {
	enum := &Enum{Name: declName(node)}
	if enum.Name == "" {
		return nil, syntaxAt(src, node, "expected identifier")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleEnumOption:
			if !leadWord(child, "option") {
				return nil, syntaxAt(src, child, "expected option")
			}
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			enum.Options = append(enum.Options, option)
		case RuleEnumReserved:
			if !leadWord(child, "reserved") {
				return nil, syntaxAt(src, child, "reserved")
			}
			reserved, err := lowerReserved(src, child)
			if err != nil {
				return nil, err
			}
			enum.Reserved = append(enum.Reserved, reserved)
		case RuleEnumValue:
			value, err := lowerEnumValue(src, child)
			if err != nil {
				return nil, err
			}
			enum.Values = append(enum.Values, value)
		}
	}
	return enum, nil
}

func lowerEnumValue(src string, node xparse.Syn[Term, Rule]) (*EnumValue, error) {
	value := &EnumValue{}
	sign := int64(1)
	sawNumber := false
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok {
			switch term {
			case TermIdentifier:
				value.Name = child.Text()
			case TermInteger:
				number, err := intToken(src, child)
				if err != nil {
					return nil, err
				}
				value.Number = int(sign * number)
				sawNumber = true
				if sign*number < math.MinInt32 || sign*number > math.MaxInt32 {
					return nil, syntaxAt(src, child, "enum number")
				}
			}
			continue
		}
		if child.Text() == "-" {
			sign = -1
			continue
		}
		if rule, ok := child.Rule(); ok && rule == RuleBrackets {
			options, err := lowerBrackets(src, child)
			if err != nil {
				return nil, err
			}
			value.Options = options
		}
	}
	if value.Name == "" || !sawNumber {
		return nil, syntaxAt(src, node, "expected enum value")
	}
	return value, nil
}

func lowerService(src string, node xparse.Syn[Term, Rule]) (*Service, error) {
	service := &Service{Name: declName(node)}
	if service.Name == "" {
		return nil, syntaxAt(src, node, "expected identifier")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleOption:
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			service.Options = append(service.Options, option)
		case RuleRPC:
			method, err := lowerRPC(src, child)
			if err != nil {
				return nil, err
			}
			service.Methods = append(service.Methods, method)
		}
	}
	return service, nil
}

func lowerRPC(src string, node xparse.Syn[Term, Rule]) (*Method, error) {
	method := &Method{}
	var types []xparse.Syn[Term, Rule]
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier && method.Name == "" {
			method.Name = child.Text()
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleRPCType:
			types = append(types, child)
		case RuleOption:
			option, err := lowerOption(src, child)
			if err != nil {
				return nil, err
			}
			method.Options = append(method.Options, option)
		}
	}
	if method.Name == "" || len(types) != 2 {
		return nil, syntaxAt(src, node, "expected rpc")
	}
	method.Input, method.ClientStream = lowerRPCType(types[0])
	method.Output, method.ServerStream = lowerRPCType(types[1])
	if method.Input == "" || method.Input == "." || method.Output == "" || method.Output == "." {
		return nil, syntaxAt(src, node, "expected type")
	}
	return method, nil
}

func lowerRPCType(node xparse.Syn[Term, Rule]) (string, bool) {
	streamed := false
	name := ""
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermStream {
			streamed = true
			continue
		}
		if rule, ok := child.Rule(); ok && rule == RuleTypeName {
			name = joinText(child)
		}
	}
	return name, streamed
}

func lowerReserved(src string, node xparse.Syn[Term, Rule]) (Reserve, error) {
	var reserved Reserve
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleStrings:
			text, err := lowerStrings(src, child)
			if err != nil {
				return Reserve{}, err
			}
			reserved.Names = append(reserved.Names, text)
		case RuleRange:
			span, err := lowerRange(src, child)
			if err != nil {
				return Reserve{}, err
			}
			reserved.Ranges = append(reserved.Ranges, span)
		}
	}
	if len(reserved.Names) == 0 && len(reserved.Ranges) == 0 {
		return Reserve{}, syntaxAt(src, node, "reserved")
	}
	return reserved, nil
}

func lowerRange(src string, node xparse.Syn[Term, Rule]) (Range, error) {
	var numbers []int
	toMax := false
	line := node
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		term, ok := child.Term()
		if !ok {
			continue
		}
		switch term {
		case TermMax:
			toMax = true
		case TermInteger:
			number, err := fieldNumber(src, child)
			if err != nil {
				return Range{}, err
			}
			if len(numbers) == 0 {
				line = child
			}
			numbers = append(numbers, number)
		}
	}
	if len(numbers) == 0 {
		return Range{}, syntaxAt(src, node, "expected field number")
	}
	span := Range{Start: numbers[0], End: numbers[0]}
	if toMax {
		span.Max = true
		span.End = 0
		return span, nil
	}
	if len(numbers) > 1 {
		span.End = numbers[1]
		if span.End < span.Start {
			return Range{}, syntaxAt(src, line, "reserved range")
		}
	}
	return span, nil
}

func lowerBrackets(src string, node xparse.Syn[Term, Rule]) ([]Option, error) {
	var options []Option
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleAssign {
			continue
		}
		option, err := lowerAssign(src, child)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, nil
}

func lowerAssign(src string, node xparse.Syn[Term, Rule]) (Option, error) {
	option := Option{}
	sawName := false
	sawValue := false
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleOptionName:
			option.Name = joinText(child)
			sawName = option.Name != ""
		case RuleConstant:
			value, err := lowerConstant(src, child)
			if err != nil {
				return Option{}, err
			}
			option.Value = value
			sawValue = true
		}
	}
	if !sawName || !sawValue {
		return Option{}, syntaxAt(src, node, "expected '='")
	}
	return option, nil
}

func lowerConstant(src string, node xparse.Syn[Term, Rule]) (Value, error) {
	if node.ChildCount() == 0 {
		return Value{}, syntaxAt(src, node, "expected constant")
	}
	switch node.Child(0).Text() {
	case "{":
		var entries []Option
		for i := 0; i < node.ChildCount(); i++ {
			child := node.Child(i)
			if rule, ok := child.Rule(); !ok || rule != RuleAssign {
				continue
			}
			entry, err := lowerAssign(src, child)
			if err != nil {
				return Value{}, err
			}
			entries = append(entries, entry)
		}
		return Aggregate(entries...), nil
	case "[":
		var items []Value
		for i := 0; i < node.ChildCount(); i++ {
			child := node.Child(i)
			if rule, ok := child.Rule(); !ok || rule != RuleConstant {
				continue
			}
			item, err := lowerConstant(src, child)
			if err != nil {
				return Value{}, err
			}
			items = append(items, item)
		}
		return List(items...), nil
	}
	sign := 0
	signNode := node
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Text() {
		case "-":
			sign = -1
			signNode = child
		case "+":
			sign = 1
			signNode = child
		}
		if rule, ok := child.Rule(); ok && rule == RuleScalar {
			return lowerScalar(src, child, sign, signNode)
		}
	}
	return Value{}, syntaxAt(src, node, "expected constant")
}

func lowerScalar(src string, node xparse.Syn[Term, Rule], sign int, signNode xparse.Syn[Term, Rule]) (Value, error) {
	if node.ChildCount() == 0 {
		return Value{}, syntaxAt(src, node, "expected constant")
	}
	child := node.Child(0)
	if rule, ok := child.Rule(); ok {
		switch rule {
		case RuleStrings:
			if sign != 0 {
				return Value{}, syntaxAt(src, signNode, "signed string")
			}
			text, err := lowerStrings(src, child)
			if err != nil {
				return Value{}, err
			}
			return String(text), nil
		case RuleFullIdentifier:
			return lowerIdent(src, child, sign, signNode)
		}
	}
	term, ok := child.Term()
	if !ok {
		return Value{}, syntaxAt(src, node, "expected constant")
	}
	switch term {
	case TermInteger:
		number, err := intToken(src, child)
		if err != nil {
			return Value{}, err
		}
		if sign < 0 {
			number = -number
		}
		return Int(number), nil
	case TermFloat:
		number, err := strconv.ParseFloat(child.Text(), 64)
		if err != nil {
			return Value{}, syntaxAt(src, child, "float")
		}
		if sign < 0 {
			number = -number
		}
		return Float(number), nil
	default:
		return Value{}, syntaxAt(src, node, "expected constant")
	}
}

func lowerIdent(src string, node xparse.Syn[Term, Rule], sign int, signNode xparse.Syn[Term, Rule]) (Value, error) {
	name := joinText(node)
	first := name
	if i := strings.IndexByte(name, '.'); i >= 0 {
		first = name[:i]
	}
	if sign != 0 && first != "inf" {
		return Value{}, syntaxAt(src, signNode, "signed "+first)
	}
	switch name {
	case "true", "false":
		if sign != 0 {
			return Value{}, syntaxAt(src, signNode, "signed "+name)
		}
		return Bool(name == "true"), nil
	case "inf":
		if sign < 0 {
			return Float(math.Inf(-1)), nil
		}
		return Float(math.Inf(1)), nil
	case "nan":
		return Float(math.NaN()), nil
	default:
		return Ident(name), nil
	}
}

func lowerType(src string, node xparse.Syn[Term, Rule]) (Type, error) {
	if rule, ok := node.Rule(); ok && rule == RuleTypeName {
		name := joinText(node)
		if name == "" || name == "." {
			return Type{}, syntaxAt(src, node, "expected type")
		}
		return Named(name), nil
	}
	var nested []Type
	head := ""
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier && head == "" {
			head = child.Text()
		}
		rule, ok := child.Rule()
		if !ok || (rule != RuleType && rule != RuleTypeName) {
			continue
		}
		typ, err := lowerType(src, child)
		if err != nil {
			return Type{}, err
		}
		nested = append(nested, typ)
	}
	switch len(nested) {
	case 1:
		return nested[0], nil
	case 2:
		if head != "map" {
			return Type{}, syntaxAt(src, node, "expected type")
		}
		return Map(nested[0], nested[1]), nil
	default:
		return Type{}, syntaxAt(src, node, "expected type")
	}
}

func lowerStrings(src string, node xparse.Syn[Term, Rule]) (string, error) {
	var text string
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		term, ok := child.Term()
		if !ok || term != TermString {
			continue
		}
		decoded, err := decodeString(child.Text())
		if err != nil {
			return "", syntaxAt(src, child, "string")
		}
		text += decoded
	}
	if text == "" && !sawString(node) {
		return "", syntaxAt(src, node, "expected string")
	}
	return text, nil
}

func sawString(node xparse.Syn[Term, Rule]) bool {
	for i := 0; i < node.ChildCount(); i++ {
		if term, ok := node.Child(i).Term(); ok && term == TermString {
			return true
		}
	}
	return false
}

func declStrings(src string, node xparse.Syn[Term, Rule]) (string, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleStrings {
			return lowerStrings(src, child)
		}
	}
	return "", syntaxAt(src, node, "expected string")
}

func leadWord(node xparse.Syn[Term, Rule], word string) bool {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier {
			return child.Text() == word
		}
	}
	return false
}

func declName(node xparse.Syn[Term, Rule]) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier {
			return child.Text()
		}
		if rule, ok := child.Rule(); ok && rule == RuleFullIdentifier {
			return joinText(child)
		}
	}
	return ""
}

func fieldNumber(src string, node xparse.Syn[Term, Rule]) (int, error) {
	number, err := intToken(src, node)
	if err != nil {
		return 0, err
	}
	if number < 1 || number > maxFieldNumber {
		return 0, syntaxAt(src, node, fmt.Sprintf("field number %d", number))
	}
	return int(number), nil
}

func intToken(src string, node xparse.Syn[Term, Rule]) (int64, error) {
	number, err := parseIntToken(node.Text())
	if err != nil {
		return 0, syntaxAt(src, node, "integer")
	}
	return number, nil
}

func joinText(node xparse.Syn[Term, Rule]) string {
	if !node.Valid() {
		return ""
	}
	if node.ChildCount() == 0 {
		return node.Text()
	}
	var b strings.Builder
	for i := 0; i < node.ChildCount(); i++ {
		b.WriteString(joinText(node.Child(i)))
	}
	return b.String()
}

func syntaxAt(src string, node xparse.Syn[Term, Rule], msg string) error {
	return syntaxError(lineOf(src, node.Start()), msg)
}

func syntaxError(line int, msg string) error {
	if line <= 0 {
		line = 1
	}
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, line, msg)
}

func lineOf(src string, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	return strings.Count(src[:off], "\n") + 1
}
