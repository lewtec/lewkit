package proto

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode"
	"unicode/utf8"
)

const maxFieldNumber = 536870911

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenIdent
	tokenString
	tokenInt
	tokenFloat
	tokenSymbol
)

type token struct {
	kind   tokenKind
	text   string
	number int64
	float  float64
	line   int
}

type parser struct {
	src  []byte
	pos  int
	line int
	buf  []token
	err  error
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
	parsed := &parser{src: body, line: 1}
	file := parsed.parseFile()
	if parsed.err != nil {
		return nil, parsed.err
	}
	return file, nil
}

func (parser *parser) parseFile() *File {
	file := NewFile()
	for parser.err == nil {
		tok := parser.peek(0)
		if tok.kind == tokenEOF {
			break
		}
		switch {
		case parser.peekSymbol(";"):
			parser.next()
		case parser.peekIdent("syntax"):
			parser.parseSyntax(file)
		case parser.peekIdent("package"):
			parser.parsePackage(file)
		case parser.peekIdent("import"):
			imported := parser.parseImport()
			if parser.err == nil {
				file.Imports = append(file.Imports, imported)
			}
		case parser.peekIdent("option"):
			file.Options = append(file.Options, parser.parseOptionStatement())
		case parser.peekIdent("message"):
			message := parser.parseMessage()
			if parser.err == nil {
				file.Messages = append(file.Messages, message)
			}
		case parser.peekIdent("enum"):
			enum := parser.parseEnum()
			if parser.err == nil {
				file.Enums = append(file.Enums, enum)
			}
		case parser.peekIdent("service"):
			service := parser.parseService()
			if parser.err == nil {
				file.Services = append(file.Services, service)
			}
		case parser.peekIdent("extend"), parser.peekIdent("extensions"):
			parser.badf(tok.line, "%s", tok.text)
			parser.next()
		default:
			parser.badf(tok.line, "unexpected %q", tok.text)
			parser.next()
		}
	}
	if parser.err != nil {
		return nil
	}
	return file
}

func (parser *parser) parseSyntax(file *File) {
	line := parser.peek(0).line
	parser.next()
	parser.expectSymbol("=")
	text := parser.parseStringConcat()
	parser.expectSymbol(";")
	if parser.err != nil {
		return
	}
	if file.Syntax != "" {
		parser.badf(line, "syntax")
		return
	}
	file.Syntax = text
}

func (parser *parser) parsePackage(file *File) {
	line := parser.peek(0).line
	parser.next()
	name := parser.parseFullIdent()
	parser.expectSymbol(";")
	if parser.err != nil {
		return
	}
	if file.Package != "" {
		parser.badf(line, "package")
		return
	}
	file.Package = name
}

func (parser *parser) parseImport() Import {
	parser.next()
	var kind ImportKind
	if parser.peekIdent("public", "weak") {
		text := parser.next().text
		parsed, err := ParseImportKind(text)
		if err != nil {
			parser.badf(parser.line, "import %s", text)
			return Import{}
		}
		kind = parsed
	}
	path := parser.parseStringConcat()
	parser.expectSymbol(";")
	return Import{Path: path, Kind: kind}
}

func (parser *parser) parseMessage() *Message {
	parser.next()
	message := &Message{Name: parser.expectIdent()}
	parser.expectSymbol("{")
	for parser.err == nil {
		tok := parser.peek(0)
		if tok.kind == tokenEOF {
			parser.badf(parser.line, "unterminated message")
			return nil
		}
		switch {
		case parser.peekSymbol("}"):
			parser.next()
			return message
		case parser.peekSymbol(";"):
			parser.next()
		case parser.peekIdent("option") && !parser.followedByEquals(1):
			message.Options = append(message.Options, parser.parseOptionStatement())
		case parser.peekIdent("reserved") && !parser.followedByEquals(1):
			message.Reserved = append(message.Reserved, parser.parseReserve())
		case parser.peekIdent("enum"):
			enum := parser.parseEnum()
			if parser.err == nil {
				message.Enums = append(message.Enums, enum)
			}
		case parser.peekIdent("message"):
			nested := parser.parseMessage()
			if parser.err == nil {
				message.Messages = append(message.Messages, nested)
			}
		case parser.peekIdent("oneof"):
			oneof := parser.parseOneof()
			if parser.err == nil {
				message.Oneofs = append(message.Oneofs, oneof)
			}
		case parser.peekIdent("extend"), parser.peekIdent("extensions"), parser.peekIdent("group"):
			parser.badf(tok.line, "%s", tok.text)
			parser.next()
		default:
			field := parser.parseField()
			if parser.err == nil {
				message.Fields = append(message.Fields, field)
			}
		}
	}
	return nil
}

func (parser *parser) parseOneof() *Oneof {
	parser.next()
	oneof := &Oneof{Name: parser.expectIdent()}
	parser.expectSymbol("{")
	for parser.err == nil {
		tok := parser.peek(0)
		if tok.kind == tokenEOF {
			parser.badf(parser.line, "unterminated oneof")
			return nil
		}
		switch {
		case parser.peekSymbol("}"):
			parser.next()
			return oneof
		case parser.peekSymbol(";"):
			parser.next()
		case parser.peekIdent("option") && !parser.followedByEquals(1):
			oneof.Options = append(oneof.Options, parser.parseOptionStatement())
		default:
			field := parser.parseField()
			if parser.err == nil {
				oneof.Fields = append(oneof.Fields, field)
			}
		}
	}
	return nil
}

func (parser *parser) parseField() *Field {
	var label Label
	if parser.peekIdent("optional", "required", "repeated") {
		text := parser.next().text
		parsed, err := ParseLabel(text)
		if err != nil {
			parser.badf(parser.line, "label %s", text)
			return nil
		}
		label = parsed
	}
	if parser.peekIdent("group") {
		parser.badf(parser.peek(0).line, "group")
		parser.next()
		return nil
	}
	typ := parser.parseType()
	name := parser.expectIdent()
	parser.expectSymbol("=")
	number := parser.parseFieldNumber()
	options := parser.parseBrackets()
	parser.expectSymbol(";")
	if parser.err != nil {
		return nil
	}
	return &Field{Name: name, Number: number, Type: typ, Label: label, Options: options}
}

func (parser *parser) parseEnum() *Enum {
	parser.next()
	enum := &Enum{Name: parser.expectIdent()}
	parser.expectSymbol("{")
	for parser.err == nil {
		tok := parser.peek(0)
		if tok.kind == tokenEOF {
			parser.badf(parser.line, "unterminated enum")
			return nil
		}
		switch {
		case parser.peekSymbol("}"):
			parser.next()
			return enum
		case parser.peekSymbol(";"):
			parser.next()
		case parser.peekIdent("option") && !parser.followedByEquals(1):
			enum.Options = append(enum.Options, parser.parseOptionStatement())
		case parser.peekIdent("reserved") && !parser.followedByEquals(1):
			enum.Reserved = append(enum.Reserved, parser.parseReserve())
		default:
			value := parser.parseEnumValue()
			if parser.err == nil {
				enum.Values = append(enum.Values, value)
			}
		}
	}
	return nil
}

func (parser *parser) parseEnumValue() *EnumValue {
	name := parser.expectIdent()
	parser.expectSymbol("=")
	number := parser.parseEnumNumber()
	options := parser.parseBrackets()
	parser.expectSymbol(";")
	if parser.err != nil {
		return nil
	}
	return &EnumValue{Name: name, Number: number, Options: options}
}

func (parser *parser) parseService() *Service {
	parser.next()
	service := &Service{Name: parser.expectIdent()}
	parser.expectSymbol("{")
	for parser.err == nil {
		tok := parser.peek(0)
		if tok.kind == tokenEOF {
			parser.badf(parser.line, "unterminated service")
			return nil
		}
		switch {
		case parser.peekSymbol("}"):
			parser.next()
			return service
		case parser.peekSymbol(";"):
			parser.next()
		case parser.peekIdent("option"):
			service.Options = append(service.Options, parser.parseOptionStatement())
		case parser.peekIdent("rpc"):
			method := parser.parseMethod()
			if parser.err == nil {
				service.Methods = append(service.Methods, method)
			}
		default:
			parser.badf(tok.line, "unexpected %q", tok.text)
			parser.next()
		}
	}
	return nil
}

func (parser *parser) parseMethod() *Method {
	parser.next()
	method := &Method{Name: parser.expectIdent()}
	parser.expectSymbol("(")
	method.Input, method.ClientStream = parser.parseRPCType()
	parser.expectSymbol(")")
	if !parser.takeIdent("returns") {
		parser.badf(parser.peek(0).line, "expected \"returns\"")
	}
	parser.expectSymbol("(")
	method.Output, method.ServerStream = parser.parseRPCType()
	parser.expectSymbol(")")
	if parser.peekSymbol("{") {
		parser.next()
		for parser.err == nil {
			if parser.peekSymbol("}") {
				parser.next()
				break
			}
			if parser.peekSymbol(";") {
				parser.next()
				continue
			}
			if parser.peek(0).kind == tokenEOF {
				parser.badf(parser.line, "unterminated rpc")
				return nil
			}
			if !parser.peekIdent("option") {
				parser.badf(parser.peek(0).line, "expected option")
				parser.next()
				return nil
			}
			method.Options = append(method.Options, parser.parseOptionStatement())
		}
	} else {
		parser.expectSymbol(";")
	}
	if parser.err != nil {
		return nil
	}
	return method
}

// parseRPCType reads an rpc argument. stream is the cardinality keyword,
// matching protoc, so stream.Foo is a stream of .Foo.
func (parser *parser) parseRPCType() (string, bool) {
	streamed := parser.takeIdent("stream")
	name := parser.parseTypeName()
	if name == "" || name == "." {
		parser.badf(parser.line, "expected type")
	}
	return name, streamed
}

func (parser *parser) parseReserve() Reserve {
	parser.next()
	if parser.peek(0).kind == tokenString {
		var names []string
		for parser.err == nil {
			names = append(names, parser.parseStringConcat())
			if parser.peekSymbol(",") {
				parser.next()
				continue
			}
			break
		}
		parser.expectSymbol(";")
		return Reserve{Names: names}
	}
	var ranges []Range
	for parser.err == nil {
		line := parser.peek(0).line
		start := parser.parseFieldNumber()
		rng := Range{Start: start, End: start}
		if parser.takeIdent("to") {
			if parser.takeIdent("max") {
				rng.Max = true
				rng.End = 0
			} else {
				rng.End = parser.parseFieldNumber()
				if parser.err == nil && rng.End < rng.Start {
					parser.badf(line, "reserved range")
				}
			}
		}
		ranges = append(ranges, rng)
		if parser.peekSymbol(",") {
			parser.next()
			continue
		}
		break
	}
	parser.expectSymbol(";")
	return Reserve{Ranges: ranges}
}

func (parser *parser) parseOptionStatement() Option {
	parser.next()
	option := parser.parseNamedConstant()
	parser.expectSymbol(";")
	return option
}

func (parser *parser) parseNamedConstant() Option {
	name := parser.parseOptionName()
	if parser.peekSymbol("=") || parser.peekSymbol(":") {
		parser.next()
	} else {
		parser.badf(parser.peek(0).line, "expected '='")
	}
	return Option{Name: name, Value: parser.parseConstant()}
}

func (parser *parser) parseOptionName() string {
	if parser.peekSymbol("(") {
		parser.next()
		name := "(" + parser.parseFullIdent() + ")"
		parser.expectSymbol(")")
		for parser.peekSymbol(".") {
			parser.next()
			name += "." + parser.expectIdent()
		}
		return name
	}
	return parser.parseFullIdent()
}

func (parser *parser) parseBrackets() []Option {
	if !parser.peekSymbol("[") {
		return nil
	}
	parser.next()
	var options []Option
	for parser.err == nil {
		if parser.peekSymbol("]") {
			parser.next()
			return options
		}
		if parser.peek(0).kind == tokenEOF {
			parser.badf(parser.line, "unterminated '['")
			return nil
		}
		options = append(options, parser.parseNamedConstant())
		if parser.peekSymbol(",") {
			parser.next()
			continue
		}
		if parser.peekSymbol("]") {
			parser.next()
			return options
		}
		parser.badf(parser.peek(0).line, "expected ']'")
		return nil
	}
	return nil
}

func (parser *parser) parseConstant() Value {
	if parser.err != nil {
		return Value{}
	}
	if parser.peekSymbol("{") {
		return parser.parseAggregate()
	}
	if parser.peekSymbol("[") {
		return parser.parseList()
	}
	sign := 0
	line := parser.peek(0).line
	if parser.peekSymbol("-") {
		sign = -1
		parser.next()
	} else if parser.peekSymbol("+") {
		sign = 1
		parser.next()
	}
	tok := parser.peek(0)
	switch tok.kind {
	case tokenString:
		if sign != 0 {
			parser.badf(line, "signed string")
			return Value{}
		}
		return String(parser.parseStringConcat())
	case tokenInt:
		parser.next()
		number := tok.number
		if sign < 0 {
			number = -number
		}
		return Int(number)
	case tokenFloat:
		parser.next()
		number := tok.float
		if sign < 0 {
			number = -number
		}
		return Float(number)
	case tokenIdent:
		if sign != 0 && tok.text != "inf" {
			parser.badf(line, "signed %s", tok.text)
			return Value{}
		}
		name := parser.parseFullIdent()
		switch name {
		case "true", "false":
			if sign != 0 {
				parser.badf(line, "signed %s", name)
				return Value{}
			}
			return Bool(name == "true")
		case "inf":
			if sign < 0 {
				return Float(math.Inf(-1))
			}
			return Float(math.Inf(1))
		case "nan":
			return Float(math.NaN())
		default:
			return Ident(name)
		}
	default:
		parser.badf(tok.line, "expected constant")
		parser.next()
		return Value{}
	}
}

func (parser *parser) parseAggregate() Value {
	parser.next()
	var entries []Option
	for parser.err == nil {
		if parser.peekSymbol("}") {
			parser.next()
			return Aggregate(entries...)
		}
		if parser.peekSymbol(",") || parser.peekSymbol(";") {
			parser.next()
			continue
		}
		if parser.peek(0).kind == tokenEOF {
			parser.badf(parser.line, "unterminated '{'")
			return Value{}
		}
		entries = append(entries, parser.parseNamedConstant())
	}
	return Value{}
}

func (parser *parser) parseList() Value {
	parser.next()
	var items []Value
	for parser.err == nil {
		if parser.peekSymbol("]") {
			parser.next()
			return List(items...)
		}
		if parser.peek(0).kind == tokenEOF {
			parser.badf(parser.line, "unterminated '['")
			return Value{}
		}
		items = append(items, parser.parseConstant())
		if parser.peekSymbol(",") {
			parser.next()
		}
	}
	return Value{}
}

func (parser *parser) parseType() Type {
	if parser.peekIdent("map") && parser.peek(1).kind == tokenSymbol && parser.peek(1).text == "<" {
		parser.next()
		parser.expectSymbol("<")
		key := parser.parseType()
		parser.expectSymbol(",")
		elem := parser.parseType()
		parser.expectSymbol(">")
		return Map(key, elem)
	}
	return Named(parser.parseTypeName())
}

func (parser *parser) parseTypeName() string {
	leadingDot := parser.peekSymbol(".")
	if leadingDot {
		parser.next()
	}
	name := parser.parseFullIdent()
	if leadingDot {
		return "." + name
	}
	return name
}

func (parser *parser) parseFullIdent() string {
	name := parser.expectIdent()
	for parser.peekSymbol(".") {
		next := parser.peek(1)
		if next.kind != tokenIdent {
			break
		}
		parser.next()
		name += "." + parser.expectIdent()
	}
	return name
}

func (parser *parser) parseStringConcat() string {
	if parser.peek(0).kind != tokenString {
		parser.badf(parser.peek(0).line, "expected string")
		return ""
	}
	text := parser.next().text
	for parser.peek(0).kind == tokenString {
		text += parser.next().text
	}
	return text
}

func (parser *parser) parseFieldNumber() int {
	tok := parser.next()
	if tok.kind != tokenInt {
		parser.badf(tok.line, "expected field number")
		return 0
	}
	if tok.number < 1 || tok.number > maxFieldNumber {
		parser.badf(tok.line, "field number %d", tok.number)
		return 0
	}
	return int(tok.number)
}

func (parser *parser) parseEnumNumber() int {
	sign := int64(1)
	if parser.peekSymbol("-") {
		sign = -1
		parser.next()
	} else if parser.peekSymbol("+") {
		parser.next()
	}
	tok := parser.next()
	if tok.kind != tokenInt {
		parser.badf(tok.line, "expected integer")
		return 0
	}
	number := sign * tok.number
	if number < math.MinInt32 || number > math.MaxInt32 {
		parser.badf(tok.line, "enum number")
		return 0
	}
	return int(number)
}

func (parser *parser) followedByEquals(n int) bool {
	tok := parser.peek(n)
	return tok.kind == tokenSymbol && tok.text == "="
}

func (parser *parser) peekIdent(names ...string) bool {
	tok := parser.peek(0)
	if tok.kind != tokenIdent {
		return false
	}
	if len(names) == 0 {
		return true
	}
	for _, name := range names {
		if tok.text == name {
			return true
		}
	}
	return false
}

func (parser *parser) takeIdent(name string) bool {
	if parser.peekIdent(name) {
		parser.next()
		return true
	}
	return false
}

func (parser *parser) peekSymbol(symbol string) bool {
	tok := parser.peek(0)
	return tok.kind == tokenSymbol && tok.text == symbol
}

func (parser *parser) expectIdent() string {
	tok := parser.next()
	if tok.kind != tokenIdent {
		parser.badf(tok.line, "expected identifier")
		return ""
	}
	return tok.text
}

func (parser *parser) expectSymbol(symbol string) {
	tok := parser.next()
	if tok.kind != tokenSymbol || tok.text != symbol {
		parser.badf(tok.line, "expected %q", symbol)
	}
}

func (parser *parser) peek(n int) token {
	for len(parser.buf) <= n {
		if parser.err != nil {
			parser.buf = append(parser.buf, token{kind: tokenEOF, line: parser.line})
			continue
		}
		parser.buf = append(parser.buf, parser.scan())
	}
	return parser.buf[n]
}

func (parser *parser) next() token {
	tok := parser.peek(0)
	parser.buf = parser.buf[1:]
	return tok
}

func (parser *parser) badf(line int, format string, args ...any) {
	if parser.err != nil {
		return
	}
	if line <= 0 {
		line = parser.line
	}
	parser.err = fmt.Errorf("%w: line %d: %s", ErrSyntax, line, fmt.Sprintf(format, args...))
}

func (parser *parser) scan() token {
	parser.skipSpace()
	if parser.err != nil || parser.pos >= len(parser.src) {
		return token{kind: tokenEOF, line: parser.line}
	}
	line := parser.line
	char := parser.src[parser.pos]
	switch {
	case isIdentStart(char):
		return parser.scanIdent(line)
	case char == '"' || char == '\'':
		return parser.scanString(line, char)
	case char >= '0' && char <= '9':
		return parser.scanNumber(line)
	case char == '.':
		if parser.pos+1 < len(parser.src) && isDigit(parser.src[parser.pos+1]) {
			return parser.scanFloat(line, parser.pos)
		}
		parser.pos++
		return token{kind: tokenSymbol, text: ".", line: line}
	case stringsContainsSymbol(char):
		parser.pos++
		return token{kind: tokenSymbol, text: string(char), line: line}
	default:
		parser.badf(line, "unexpected %q", char)
		parser.pos++
		return token{kind: tokenEOF, line: line}
	}
}

func stringsContainsSymbol(char byte) bool {
	switch char {
	case '=', '{', '}', '(', ')', '[', ']', '<', '>', ',', ';', ':', '+', '-':
		return true
	default:
		return false
	}
}

func (parser *parser) skipSpace() {
	for parser.pos < len(parser.src) && parser.err == nil {
		char := parser.src[parser.pos]
		switch char {
		case ' ', '\t', '\r', '\v', '\f':
			parser.pos++
		case '\n':
			parser.pos++
			parser.line++
		case '/':
			if parser.pos+1 >= len(parser.src) {
				return
			}
			switch parser.src[parser.pos+1] {
			case '/':
				parser.pos += 2
				for parser.pos < len(parser.src) && parser.src[parser.pos] != '\n' {
					parser.pos++
				}
			case '*':
				parser.skipBlock()
			default:
				return
			}
		default:
			return
		}
	}
}

func (parser *parser) skipBlock() {
	parser.pos += 2
	for parser.pos+1 < len(parser.src) {
		if parser.src[parser.pos] == '\n' {
			parser.line++
		}
		if parser.src[parser.pos] == '*' && parser.src[parser.pos+1] == '/' {
			parser.pos += 2
			return
		}
		parser.pos++
	}
	parser.badf(parser.line, "unterminated comment")
}

func (parser *parser) scanIdent(line int) token {
	start := parser.pos
	parser.pos++
	for parser.pos < len(parser.src) && isIdentCont(parser.src[parser.pos]) {
		parser.pos++
	}
	return token{kind: tokenIdent, text: string(parser.src[start:parser.pos]), line: line}
}

func (parser *parser) scanString(line int, quote byte) token {
	parser.pos++
	var text []byte
	for parser.pos < len(parser.src) {
		char := parser.src[parser.pos]
		if char == '\n' || char == '\r' {
			parser.badf(line, "newline in string")
			return token{kind: tokenEOF, line: line}
		}
		if char == quote {
			parser.pos++
			return token{kind: tokenString, text: string(text), line: line}
		}
		if char != '\\' {
			text = append(text, char)
			parser.pos++
			continue
		}
		parser.pos++
		if parser.pos >= len(parser.src) {
			parser.badf(line, "escape")
			return token{kind: tokenEOF, line: line}
		}
		escaped := parser.src[parser.pos]
		parser.pos++
		switch escaped {
		case 'a':
			text = append(text, '\a')
		case 'b':
			text = append(text, '\b')
		case 'f':
			text = append(text, '\f')
		case 'n':
			text = append(text, '\n')
		case 'r':
			text = append(text, '\r')
		case 't':
			text = append(text, '\t')
		case 'v':
			text = append(text, '\v')
		case '\\', '\'', '"':
			text = append(text, escaped)
		case 'x', 'X':
			value, ok := parser.readHex(line, 1, 2)
			if !ok {
				return token{kind: tokenEOF, line: line}
			}
			text = append(text, byte(value))
		case 'u':
			value, ok := parser.readHex(line, 4, 4)
			if !ok {
				return token{kind: tokenEOF, line: line}
			}
			text = appendRune(text, rune(value), line, parser)
		case 'U':
			value, ok := parser.readHex(line, 8, 8)
			if !ok {
				return token{kind: tokenEOF, line: line}
			}
			text = appendRune(text, rune(value), line, parser)
		case '0', '1', '2', '3', '4', '5', '6', '7':
			octal := int(escaped - '0')
			digits := 1
			for digits < 3 && parser.pos < len(parser.src) && parser.src[parser.pos] >= '0' && parser.src[parser.pos] <= '7' {
				octal = octal*8 + int(parser.src[parser.pos]-'0')
				parser.pos++
				digits++
			}
			text = append(text, byte(octal))
		default:
			parser.badf(line, "escape")
			return token{kind: tokenEOF, line: line}
		}
		if parser.err != nil {
			return token{kind: tokenEOF, line: line}
		}
	}
	parser.badf(line, "unterminated string")
	return token{kind: tokenEOF, line: line}
}

func appendRune(text []byte, char rune, line int, parser *parser) []byte {
	if char > unicode.MaxRune {
		parser.badf(line, "rune")
		return text
	}
	encoded := make([]byte, utf8.UTFMax)
	size := utf8.EncodeRune(encoded, char)
	return append(text, encoded[:size]...)
}

func (parser *parser) readHex(line, minDigits, maxDigits int) (int, bool) {
	value := 0
	digits := 0
	for digits < maxDigits && parser.pos < len(parser.src) && isHex(parser.src[parser.pos]) {
		value = value<<4 | hexValue(parser.src[parser.pos])
		parser.pos++
		digits++
	}
	if digits < minDigits {
		parser.badf(line, "hex escape")
		return 0, false
	}
	return value, true
}

func (parser *parser) scanNumber(line int) token {
	start := parser.pos
	if parser.src[parser.pos] == '0' && parser.pos+1 < len(parser.src) && (parser.src[parser.pos+1] == 'x' || parser.src[parser.pos+1] == 'X') {
		parser.pos += 2
		hexStart := parser.pos
		for parser.pos < len(parser.src) && isHex(parser.src[parser.pos]) {
			parser.pos++
		}
		if parser.pos == hexStart {
			parser.badf(line, "hex digit")
			return token{kind: tokenEOF, line: line}
		}
		number, err := strconv.ParseInt(string(parser.src[hexStart:parser.pos]), 16, 64)
		if err != nil {
			parser.badf(line, "integer")
			return token{kind: tokenEOF, line: line}
		}
		return token{kind: tokenInt, text: string(parser.src[start:parser.pos]), number: number, line: line}
	}
	if parser.src[parser.pos] == '0' {
		index := parser.pos + 1
		octal := true
		sawDigit := false
		for index < len(parser.src) && isDigit(parser.src[index]) {
			sawDigit = true
			if parser.src[index] > '7' {
				octal = false
			}
			index++
		}
		floatish := index < len(parser.src) && (parser.src[index] == '.' || parser.src[index] == 'e' || parser.src[index] == 'E')
		if floatish {
			return parser.scanFloat(line, start)
		}
		if sawDigit && !octal {
			parser.badf(line, "octal digit")
			parser.pos = index
			return token{kind: tokenEOF, line: line}
		}
		parser.pos = index
		number, err := strconv.ParseInt(string(parser.src[start:parser.pos]), 8, 64)
		if err != nil {
			parser.badf(line, "integer")
			return token{kind: tokenEOF, line: line}
		}
		return token{kind: tokenInt, text: string(parser.src[start:parser.pos]), number: number, line: line}
	}
	for parser.pos < len(parser.src) && isDigit(parser.src[parser.pos]) {
		parser.pos++
	}
	if parser.pos < len(parser.src) && (parser.src[parser.pos] == '.' || parser.src[parser.pos] == 'e' || parser.src[parser.pos] == 'E') {
		return parser.scanFloat(line, start)
	}
	number, err := strconv.ParseInt(string(parser.src[start:parser.pos]), 10, 64)
	if err != nil {
		parser.badf(line, "integer")
		return token{kind: tokenEOF, line: line}
	}
	return token{kind: tokenInt, text: string(parser.src[start:parser.pos]), number: number, line: line}
}

func (parser *parser) scanFloat(line, start int) token {
	parser.pos = start
	if parser.pos < len(parser.src) && parser.src[parser.pos] == '.' {
		parser.pos++
		if parser.pos >= len(parser.src) || !isDigit(parser.src[parser.pos]) {
			parser.badf(line, "float")
			return token{kind: tokenEOF, line: line}
		}
		for parser.pos < len(parser.src) && isDigit(parser.src[parser.pos]) {
			parser.pos++
		}
	} else {
		for parser.pos < len(parser.src) && isDigit(parser.src[parser.pos]) {
			parser.pos++
		}
		if parser.pos < len(parser.src) && parser.src[parser.pos] == '.' {
			parser.pos++
			for parser.pos < len(parser.src) && isDigit(parser.src[parser.pos]) {
				parser.pos++
			}
		}
	}
	if parser.pos < len(parser.src) && (parser.src[parser.pos] == 'e' || parser.src[parser.pos] == 'E') {
		parser.pos++
		if parser.pos < len(parser.src) && (parser.src[parser.pos] == '+' || parser.src[parser.pos] == '-') {
			parser.pos++
		}
		if parser.pos >= len(parser.src) || !isDigit(parser.src[parser.pos]) {
			parser.badf(line, "exponent")
			return token{kind: tokenEOF, line: line}
		}
		for parser.pos < len(parser.src) && isDigit(parser.src[parser.pos]) {
			parser.pos++
		}
	}
	text := string(parser.src[start:parser.pos])
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		parser.badf(line, "float")
		return token{kind: tokenEOF, line: line}
	}
	return token{kind: tokenFloat, text: text, float: number, line: line}
}

func isIdentStart(char byte) bool {
	return char == '_' || (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
}

func isIdentCont(char byte) bool {
	return isIdentStart(char) || isDigit(char)
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isHex(char byte) bool {
	return isDigit(char) || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

func hexValue(char byte) int {
	switch {
	case char >= '0' && char <= '9':
		return int(char - '0')
	case char >= 'a' && char <= 'f':
		return int(char-'a') + 10
	default:
		return int(char-'A') + 10
	}
}
