package sql

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func parse(text string) (*File, error) {
	p := &parser{lex: lexer{src: text, line: 1}, file: NewFile()}
	p.next()
	p.applyDialect()
	for {
		if p.lex.err != nil {
			return nil, p.lex.err
		}
		if p.tok.kind == tokEOF {
			if !p.file.Dialect.IsValid() {
				p.file.Dialect = SQLite
			}
			return p.file, nil
		}
		if p.eatOp(";") {
			continue
		}
		comment := p.tok.comment
		p.tok.comment = ""
		stmt, err := p.statement()
		if err != nil {
			return nil, err
		}
		setComment(stmt, comment)
		p.file.Statements = append(p.file.Statements, stmt)
	}
}

func (p *parser) applyDialect() {
	rest, dialect, ok := splitDialect(p.tok.comment)
	if ok {
		p.file.Dialect = dialect
		p.tok.comment = rest
		return
	}
	if !p.file.Dialect.IsValid() {
		p.file.Dialect = SQLite
	}
}

func splitDialect(comment string) (string, Dialect, bool) {
	if comment == "" {
		return "", 0, false
	}
	line, tail, _ := strings.Cut(comment, "\n")
	key, val, found := strings.Cut(strings.TrimSpace(line), ":")
	if !found || !strings.EqualFold(strings.TrimSpace(key), "dialect") {
		return comment, 0, false
	}
	dialect, err := ParseDialect(strings.TrimSpace(val))
	if err != nil {
		return comment, 0, false
	}
	return strings.TrimSpace(tail), dialect, true
}

type parser struct {
	lex  lexer
	tok  token
	file *File
}

func (p *parser) next() { p.tok = p.lex.next() }

func (p *parser) fail(msg string) error {
	if p.lex.err != nil {
		return p.lex.err
	}
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, p.tok.line, msg)
}

func (p *parser) keyword(word string) bool {
	return p.tok.kind == tokIdent && !p.tok.quoted && strings.EqualFold(p.tok.text, word)
}

func (p *parser) eat(word string) bool {
	if !p.keyword(word) {
		return false
	}
	p.next()
	return true
}

func (p *parser) eatOp(op string) bool {
	if p.tok.kind != tokOp || p.tok.text != op {
		return false
	}
	p.next()
	return true
}

func (p *parser) want(word string) error {
	if p.eat(word) {
		return nil
	}
	return p.fail("expected " + word)
}

func (p *parser) wantOp(op string) error {
	if p.eatOp(op) {
		return nil
	}
	return p.fail("expected " + op)
}

func (p *parser) simpleName() (string, error) {
	if p.tok.kind != tokIdent {
		return "", p.fail("expected name")
	}
	name := p.tok.text
	p.next()
	return name, nil
}

func (p *parser) nameList() ([]string, error) {
	name, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	names := []string{name}
	for p.eatOp(",") {
		name, err = p.simpleName()
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func (p *parser) parenNames() ([]string, error) {
	if err := p.wantOp("("); err != nil {
		return nil, err
	}
	names, err := p.nameList()
	if err != nil {
		return nil, err
	}
	if err := p.wantOp(")"); err != nil {
		return nil, err
	}
	return names, nil
}

func (p *parser) qualified() (string, error) {
	name, err := p.simpleName()
	if err != nil {
		return "", err
	}
	for p.eatOp(".") {
		rest, err := p.simpleName()
		if err != nil {
			return "", err
		}
		name += "." + rest
	}
	return name, nil
}

func (p *parser) expr() (Expr, error) { return p.or() }

func (p *parser) or() (Expr, error) {
	left, err := p.and()
	if err != nil {
		return Expr{}, err
	}
	for p.eat("or") {
		right, err := p.and()
		if err != nil {
			return Expr{}, err
		}
		left = binary("OR", left, right)
	}
	return left, nil
}

func (p *parser) and() (Expr, error) {
	left, err := p.not()
	if err != nil {
		return Expr{}, err
	}
	for p.eat("and") {
		right, err := p.not()
		if err != nil {
			return Expr{}, err
		}
		left = binary("AND", left, right)
	}
	return left, nil
}

func (p *parser) not() (Expr, error) {
	if p.eat("not") {
		expr, err := p.not()
		if err != nil {
			return Expr{}, err
		}
		return unary("NOT", expr), nil
	}
	return p.cmp()
}

func (p *parser) cmp() (Expr, error) {
	left, err := p.sum()
	if err != nil {
		return Expr{}, err
	}
	for {
		op := ""
		switch {
		case p.eatOp("="):
			op = "="
		case p.eatOp("<>"):
			op = "<>"
		case p.eatOp("!="):
			op = "!="
		case p.eatOp("<="):
			op = "<="
		case p.eatOp(">="):
			op = ">="
		case p.eatOp("<"):
			op = "<"
		case p.eatOp(">"):
			op = ">"
		case p.eat("like"):
			op = "LIKE"
		case p.eat("is"):
			not := p.eat("not")
			if err := p.want("null"); err != nil {
				return Expr{}, err
			}
			name := "IS"
			if not {
				name = "IS NOT"
			}
			left = binary(name, left, Id("NULL"))
			continue
		default:
			return left, nil
		}
		right, err := p.sum()
		if err != nil {
			return Expr{}, err
		}
		left = binary(op, left, right)
	}
}

func (p *parser) sum() (Expr, error) {
	left, err := p.product()
	if err != nil {
		return Expr{}, err
	}
	for {
		op := ""
		switch {
		case p.eatOp("+"):
			op = "+"
		case p.eatOp("-"):
			op = "-"
		case p.eatOp("||"):
			op = "||"
		default:
			return left, nil
		}
		right, err := p.product()
		if err != nil {
			return Expr{}, err
		}
		left = binary(op, left, right)
	}
}

func (p *parser) product() (Expr, error) {
	left, err := p.unary()
	if err != nil {
		return Expr{}, err
	}
	for {
		op := ""
		switch {
		case p.eatOp("*"):
			op = "*"
		case p.eatOp("/"):
			op = "/"
		case p.eatOp("%"):
			op = "%"
		default:
			return left, nil
		}
		right, err := p.unary()
		if err != nil {
			return Expr{}, err
		}
		left = binary(op, left, right)
	}
}

func (p *parser) unary() (Expr, error) {
	if p.eatOp("-") {
		expr, err := p.unary()
		if err != nil {
			return Expr{}, err
		}
		return unary("-", expr), nil
	}
	if p.eatOp("+") {
		expr, err := p.unary()
		if err != nil {
			return Expr{}, err
		}
		return unary("+", expr), nil
	}
	return p.primary()
}

func (p *parser) primary() (Expr, error) {
	switch p.tok.kind {
	case tokString:
		expr := Lit(p.tok.text)
		p.next()
		return expr, nil
	case tokNumber:
		expr := Num(p.tok.text)
		p.next()
		return expr, nil
	case tokPlaceholder:
		expr := Expr{Kind: ExprKindPlaceholder, Text: p.tok.text}
		p.next()
		return expr, nil
	case tokOp:
		if p.eatOp("*") {
			return Star(), nil
		}
		if p.eatOp("(") {
			expr, err := p.expr()
			if err != nil {
				return Expr{}, err
			}
			if err := p.wantOp(")"); err != nil {
				return Expr{}, err
			}
			return expr, nil
		}
		return Expr{}, p.fail("expected expression")
	case tokIdent:
		name, err := p.qualified()
		if err != nil {
			return Expr{}, err
		}
		if !p.eatOp("(") {
			return Id(name), nil
		}
		var args []Expr
		if !p.atOp(")") {
			for {
				arg, err := p.expr()
				if err != nil {
					return Expr{}, err
				}
				args = append(args, arg)
				if !p.eatOp(",") {
					break
				}
			}
		}
		if err := p.wantOp(")"); err != nil {
			return Expr{}, err
		}
		return Call(name, args...), nil
	default:
		return Expr{}, p.fail("expected expression")
	}
}

func (p *parser) atOp(op string) bool {
	return p.tok.kind == tokOp && p.tok.text == op
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokNumber
	tokPlaceholder
	tokOp
)

type token struct {
	kind    tokenKind
	text    string
	quoted  bool
	line    int
	comment string
}

type lexer struct {
	src     string
	i       int
	line    int
	comment strings.Builder
	err     error
}

func (l *lexer) next() token {
	if l.err != nil {
		return token{kind: tokEOF, line: l.line}
	}
	l.skip()
	if l.err != nil {
		return token{kind: tokEOF, line: l.line}
	}
	comment := strings.TrimSpace(l.comment.String())
	l.comment.Reset()
	if l.i >= len(l.src) {
		return token{kind: tokEOF, line: l.line, comment: comment}
	}
	line := l.line
	c := l.src[l.i]
	switch c {
	case '\'':
		text, ok := l.string()
		if !ok {
			return l.broken(line, "unclosed string")
		}
		return token{kind: tokString, text: text, line: line, comment: comment}
	case '"', '`', '[':
		text, ok := l.quoted(c)
		if !ok {
			return l.broken(line, "unclosed identifier")
		}
		return token{kind: tokIdent, text: text, quoted: true, line: line, comment: comment}
	case '?':
		l.i++
		return token{kind: tokPlaceholder, text: "?", line: line, comment: comment}
	case '$':
		if l.i+1 < len(l.src) && isDigit(l.src[l.i+1]) {
			start := l.i
			l.i++
			for l.i < len(l.src) && isDigit(l.src[l.i]) {
				l.i++
			}
			return token{kind: tokPlaceholder, text: l.src[start:l.i], line: line, comment: comment}
		}
		return l.broken(line, "expected placeholder")
	default:
		if isIdentStart(c) {
			start := l.i
			l.i++
			for l.i < len(l.src) && isIdentCont(l.src[l.i]) {
				l.i++
			}
			return token{kind: tokIdent, text: l.src[start:l.i], line: line, comment: comment}
		}
		if isDigit(c) {
			return token{kind: tokNumber, text: l.number(), line: line, comment: comment}
		}
		op, ok := l.operator()
		if !ok {
			return l.broken(line, "unexpected character")
		}
		return token{kind: tokOp, text: op, line: line, comment: comment}
	}
}

func (l *lexer) broken(line int, msg string) token {
	if l.err == nil {
		l.err = fmt.Errorf("%w: line %d: %s", ErrSyntax, line, msg)
	}
	return token{kind: tokEOF, line: line}
}

func (l *lexer) skip() {
	for l.i < len(l.src) && l.err == nil {
		c := l.src[l.i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			l.bump()
			continue
		}
		if c == '-' && l.i+1 < len(l.src) && l.src[l.i+1] == '-' {
			l.lineComment()
			continue
		}
		if c == '/' && l.i+1 < len(l.src) && l.src[l.i+1] == '*' {
			l.blockComment()
			continue
		}
		return
	}
}

func (l *lexer) lineComment() {
	l.i += 2
	start := l.i
	for l.i < len(l.src) && l.src[l.i] != '\n' && l.src[l.i] != '\r' {
		l.i++
	}
	l.addComment(strings.TrimSpace(l.src[start:l.i]))
}

func (l *lexer) blockComment() {
	l.i += 2
	start := l.i
	for l.i+1 < len(l.src) {
		if l.src[l.i] == '*' && l.src[l.i+1] == '/' {
			l.addComment(strings.TrimSpace(l.src[start:l.i]))
			l.i += 2
			return
		}
		if l.src[l.i] == '\n' {
			l.line++
		} else if l.src[l.i] == '\r' {
			l.line++
			if l.src[l.i+1] == '\n' {
				l.i++
			}
		}
		l.i++
	}
	l.broken(l.line, "unclosed comment")
}

func (l *lexer) addComment(text string) {
	if text == "" {
		return
	}
	if l.comment.Len() > 0 {
		l.comment.WriteByte('\n')
	}
	l.comment.WriteString(text)
}

func (l *lexer) bump() {
	if l.i >= len(l.src) {
		return
	}
	c := l.src[l.i]
	l.i++
	if c == '\n' {
		l.line++
		return
	}
	if c == '\r' {
		l.line++
		if l.i < len(l.src) && l.src[l.i] == '\n' {
			l.i++
		}
	}
}

func (l *lexer) string() (string, bool) {
	l.i++
	var b strings.Builder
	for l.i < len(l.src) {
		c := l.src[l.i]
		if c == '\n' || c == '\r' {
			l.bump()
			b.WriteByte('\n')
			continue
		}
		l.i++
		if c != '\'' {
			b.WriteByte(c)
			continue
		}
		if l.i < len(l.src) && l.src[l.i] == '\'' {
			b.WriteByte('\'')
			l.i++
			continue
		}
		return b.String(), true
	}
	return "", false
}

func (l *lexer) quoted(quote byte) (string, bool) {
	l.i++
	end := quote
	if quote == '[' {
		end = ']'
	}
	var b strings.Builder
	for l.i < len(l.src) {
		c := l.src[l.i]
		if c == '\n' || c == '\r' {
			l.bump()
			b.WriteByte('\n')
			continue
		}
		if c == end {
			if quote != '[' && l.i+1 < len(l.src) && l.src[l.i+1] == end {
				b.WriteByte(end)
				l.i += 2
				continue
			}
			l.i++
			return b.String(), true
		}
		b.WriteByte(c)
		l.i++
	}
	return "", false
}

func (l *lexer) number() string {
	start := l.i
	for l.i < len(l.src) && isDigit(l.src[l.i]) {
		l.i++
	}
	if l.i+1 < len(l.src) && l.src[l.i] == '.' && isDigit(l.src[l.i+1]) {
		l.i++
		for l.i < len(l.src) && isDigit(l.src[l.i]) {
			l.i++
		}
	}
	if l.i < len(l.src) && (l.src[l.i] == 'e' || l.src[l.i] == 'E') {
		j := l.i + 1
		if j < len(l.src) && (l.src[j] == '+' || l.src[j] == '-') {
			j++
		}
		if j < len(l.src) && isDigit(l.src[j]) {
			l.i = j
			for l.i < len(l.src) && isDigit(l.src[l.i]) {
				l.i++
			}
		}
	}
	return l.src[start:l.i]
}

func (l *lexer) operator() (string, bool) {
	if l.i+1 < len(l.src) {
		pair := l.src[l.i : l.i+2]
		switch pair {
		case "<>", "!=", "<=", ">=", "||":
			l.i += 2
			return pair, true
		}
	}
	switch l.src[l.i] {
	case '(', ')', ',', ';', '.', '*', '=', '<', '>', '+', '-', '/', '%':
		op := l.src[l.i : l.i+1]
		l.i++
		return op, true
	default:
		return "", false
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c >= utf8.RuneSelf
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

func (p *parser) sqlType() (Type, error) {
	if p.tok.kind != tokIdent || p.tok.quoted {
		return 0, p.fail("expected type")
	}
	word := p.tok.text
	p.next()
	if p.atOp("(") {
		return 0, p.fail("type parameters are unsupported")
	}
	switch strings.ToUpper(word) {
	case "INTEGER", "INT":
		return TypeInteger, nil
	case "TEXT":
		return TypeText, nil
	case "BLOB":
		return TypeBlob, nil
	case "REAL":
		return TypeReal, nil
	case "NUMERIC":
		return TypeNumeric, nil
	case "BIGINT":
		return TypeBigInt, nil
	case "BOOLEAN", "BOOL":
		return TypeBoolean, nil
	case "TIMESTAMP":
		return TypeTimestamp, nil
	default:
		return 0, p.fail("unknown type " + word)
	}
}
