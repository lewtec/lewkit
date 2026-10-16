package llvm

import (
	"fmt"
	"strings"
)

type tokKind int

const (
	tokEOF tokKind = iota
	tokIdent
	tokNumber
	tokString
	tokCString
	tokLocal
	tokGlobal
	tokDots
	tokPunct
)

type token struct {
	kind  tokKind
	text  string
	line  int
	col   int
	start int
	end   int
}

type lexer struct {
	s    string
	i    int
	line int
	col  int
	err  error
}

func newLexer(s string) *lexer {
	return &lexer{s: s, line: 1, col: 1}
}

func (l *lexer) next() token {
	l.skip()
	if l.err != nil || l.i >= len(l.s) {
		return token{kind: tokEOF, line: l.line, col: l.col, start: l.i, end: l.i}
	}
	start := l.i
	line, col := l.line, l.col
	tok := l.scan(line, col)
	tok.start = start
	tok.end = l.i
	tok.line = line
	tok.col = col
	return tok
}

func (l *lexer) scan(line, col int) token {
	c := l.s[l.i]
	switch c {
	case '"':
		return l.lexString(false, line, col)
	case '%', '@':
		return l.lexSigil(c, line, col)
	case ',', '=', '(', ')', '{', '}', '[', ']', '<', '>', '*', '#', ':', '!':
		l.bump()
		return token{kind: tokPunct, text: string(c), line: line, col: col}
	case '.':
		if l.peekN(1) == '.' && l.peekN(2) == '.' {
			l.bump()
			l.bump()
			l.bump()
			return token{kind: tokDots, text: "...", line: line, col: col}
		}
		if isDigit(l.peekN(1)) {
			return l.lexNumber(line, col)
		}
	}
	if c == 'c' && l.peekN(1) == '"' {
		return l.lexString(true, line, col)
	}
	if c == '-' && l.numberAfterDash() {
		return l.lexNumber(line, col)
	}
	if isDigit(c) {
		return l.lexNumber(line, col)
	}
	if isIdentStart(c) {
		return l.lexIdent(line, col)
	}
	l.fail(line, col, "bad character")
	return token{kind: tokEOF, line: line, col: col}
}

func (l *lexer) numberAfterDash() bool {
	n := l.peekN(1)
	if isDigit(n) {
		return true
	}
	if n == '.' && isDigit(l.peekN(2)) {
		return true
	}
	return false
}

func (l *lexer) skip() {
	for l.i < len(l.s) && l.err == nil {
		c := l.s[l.i]
		if c == ';' {
			for l.i < len(l.s) && l.s[l.i] != '\n' {
				l.bump()
			}
			continue
		}
		if isSpace(c) {
			l.bump()
			continue
		}
		return
	}
}

func (l *lexer) lexIdent(line, col int) token {
	start := l.i
	l.bump()
	for isIdentCont(l.peek()) {
		l.bump()
	}
	return token{kind: tokIdent, text: l.s[start:l.i], line: line, col: col}
}

func (l *lexer) lexNumber(line, col int) token {
	start := l.i
	if l.peek() == '-' {
		l.bump()
	}
	if l.peek() == '0' && (l.peekN(1) == 'x' || l.peekN(1) == 'X') {
		l.bump()
		l.bump()
		// K L M H R mark float, fp80, fp128, ppc_fp128, half, and bfloat.
		switch l.peek() {
		case 'K', 'L', 'M', 'H', 'R':
			l.bump()
		}
		saw := false
		for isHex(l.peek()) {
			saw = true
			l.bump()
		}
		if l.peek() == '.' && isHex(l.peekN(1)) {
			l.bump()
			for isHex(l.peek()) {
				saw = true
				l.bump()
			}
		}
		if l.peek() == 'p' || l.peek() == 'P' {
			l.bump()
			if l.peek() == '+' || l.peek() == '-' {
				l.bump()
			}
			if !isDigit(l.peek()) {
				l.fail(line, col, "bad exponent")
				return token{kind: tokEOF, line: line, col: col}
			}
			for isDigit(l.peek()) {
				l.bump()
			}
			saw = true
		}
		if !saw {
			l.fail(line, col, "bad hex")
			return token{kind: tokEOF, line: line, col: col}
		}
		return token{kind: tokNumber, text: l.s[start:l.i], line: line, col: col}
	}
	sawDigit := false
	for isDigit(l.peek()) {
		sawDigit = true
		l.bump()
	}
	if l.peek() == '.' && isDigit(l.peekN(1)) {
		l.bump()
		for isDigit(l.peek()) {
			sawDigit = true
			l.bump()
		}
	}
	if l.peek() == 'e' || l.peek() == 'E' {
		l.bump()
		if l.peek() == '+' || l.peek() == '-' {
			l.bump()
		}
		if !isDigit(l.peek()) {
			l.fail(line, col, "bad exponent")
			return token{kind: tokEOF, line: line, col: col}
		}
		for isDigit(l.peek()) {
			l.bump()
		}
		sawDigit = true
	}
	if !sawDigit {
		l.fail(line, col, "bad number")
		return token{kind: tokEOF, line: line, col: col}
	}
	return token{kind: tokNumber, text: l.s[start:l.i], line: line, col: col}
}

func (l *lexer) lexSigil(sig byte, line, col int) token {
	l.bump()
	kind := tokLocal
	if sig == '@' {
		kind = tokGlobal
	}
	if l.peek() == '"' {
		tok := l.lexString(false, line, col)
		if l.err != nil {
			return token{kind: tokEOF, line: line, col: col}
		}
		tok.kind = kind
		tok.line = line
		tok.col = col
		return tok
	}
	if isDigit(l.peek()) {
		start := l.i
		for isDigit(l.peek()) {
			l.bump()
		}
		return token{kind: kind, text: l.s[start:l.i], line: line, col: col}
	}
	if !isSigilStart(l.peek()) {
		l.fail(line, col, "bad name")
		return token{kind: tokEOF, line: line, col: col}
	}
	start := l.i
	for isSigilCont(l.peek()) {
		l.bump()
	}
	return token{kind: kind, text: l.s[start:l.i], line: line, col: col}
}

func (l *lexer) lexString(prefix bool, line, col int) token {
	if prefix {
		l.bump()
	}
	if l.peek() != '"' {
		l.fail(line, col, "string")
		return token{kind: tokEOF, line: line, col: col}
	}
	l.bump()
	var b strings.Builder
	for l.i < len(l.s) {
		c := l.s[l.i]
		if c == '"' {
			l.bump()
			kind := tokString
			if prefix {
				kind = tokCString
			}
			return token{kind: kind, text: b.String(), line: line, col: col}
		}
		if c == '\n' {
			break
		}
		if c == '\\' {
			l.bump()
			if l.i >= len(l.s) {
				break
			}
			n := l.s[l.i]
			if n == '\\' || n == '"' {
				b.WriteByte(n)
				l.bump()
				continue
			}
			if l.i+1 >= len(l.s) {
				break
			}
			hi := unhex(l.s[l.i])
			lo := unhex(l.s[l.i+1])
			if hi < 0 || lo < 0 {
				l.fail(line, col, "bad escape")
				return token{kind: tokEOF, line: line, col: col}
			}
			b.WriteByte(byte(hi<<4 | lo))
			l.bump()
			l.bump()
			continue
		}
		b.WriteByte(c)
		l.bump()
	}
	l.fail(line, col, "unterminated string")
	return token{kind: tokEOF, line: line, col: col}
}

func (l *lexer) fail(line, col int, what string) {
	if l.err == nil {
		l.err = fmt.Errorf("%w: %d:%d: %s", ErrSyntax, line, col, what)
	}
}

func (l *lexer) bump() {
	if l.i >= len(l.s) {
		return
	}
	if l.s[l.i] == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.i++
}

func (l *lexer) peek() byte { return l.peekN(0) }

func (l *lexer) peekN(n int) byte {
	if l.i+n >= len(l.s) {
		return 0
	}
	return l.s[l.i+n]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentStart(c byte) bool {
	return isLetter(c) || c == '$' || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || isDigit(c) || c == '.' || c == '-'
}

func isSigilStart(c byte) bool {
	return isIdentStart(c) || c == '.' || c == '-'
}

func isSigilCont(c byte) bool {
	return isSigilStart(c) || isDigit(c)
}

func isHex(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
