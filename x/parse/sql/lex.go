package sql

import (
	"strings"
	"unicode/utf8"

	xparse "github.com/lewtec/lewkit/x/parse"
)

func matchKeyword(text string) xparse.Match {
	return func(src []byte, at int) int {
		if at+len(text) > len(src) {
			return 0
		}
		for i := 0; i < len(text); i++ {
			if foldASCII(src[at+i]) != foldASCII(text[i]) {
				return 0
			}
		}
		end := at + len(text)
		if end < len(src) {
			r, _ := utf8.DecodeRune(src[end:])
			if identifierContinue(r) {
				return 0
			}
		}
		return len(text)
	}
}

func foldASCII(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

func identifierStart(r rune) bool {
	if r >= utf8.RuneSelf {
		return true
	}
	return r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func identifierContinue(r rune) bool {
	return identifierStart(r) || (r >= '0' && r <= '9')
}

// MatchIdentBytes reports how many bytes at at are an identifier, or 0.
func MatchIdentBytes(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	r, n := utf8.DecodeRune(src[at:])
	if r == utf8.RuneError && n == 1 || !identifierStart(r) {
		return 0
	}
	i := at + n
	for i < len(src) {
		r, n = utf8.DecodeRune(src[i:])
		if r == utf8.RuneError && n == 1 || !identifierContinue(r) {
			break
		}
		i += n
	}
	return i - at
}

// matchAlias is a bare select alias. Clause words stay available as
// clauses; the lexer would otherwise keep the lower symbol on a tie.
func matchAlias(src []byte, at int) int {
	n := MatchIdentBytes(src, at)
	if n == 0 {
		return 0
	}
	switch strings.ToUpper(string(src[at : at+n])) {
	case "FROM", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING", "UNION", "EXCEPT", "INTERSECT", "WINDOW":
		return 0
	default:
		return n
	}
}

func matchLineComment(src []byte, at int) int {
	if at+1 >= len(src) || src[at] != '-' || src[at+1] != '-' {
		return 0
	}
	i := at + 2
	for i < len(src) && src[i] != '\n' && src[i] != '\r' {
		i++
	}
	return i - at
}

func matchBlockComment(src []byte, at int) int {
	if at+1 >= len(src) || src[at] != '/' || src[at+1] != '*' {
		return 0
	}
	end := blockEnd(string(src), at)
	if end < 0 {
		return 0
	}
	return end + 2 - at
}

func blockEnd(src string, at int) int {
	for i := at + 2; i+1 < len(src); i++ {
		if src[i] == '*' && src[i+1] == '/' {
			return i
		}
	}
	return -1
}

// MatchString reports how many bytes at at are one SQL string, or 0.
func MatchString(src []byte, at int) int {
	if at >= len(src) || src[at] != '\'' {
		return 0
	}
	i := at + 1
	for i < len(src) {
		if src[i] == '\'' {
			if i+1 < len(src) && src[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1 - at
		}
		i++
	}
	return 0
}

// MatchQuoted reports how many bytes at at are a quoted identifier, or 0.
func MatchQuoted(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	quote := src[at]
	end := quote
	switch quote {
	case '"', '`':
	case '[':
		end = ']'
	default:
		return 0
	}
	i := at + 1
	for i < len(src) {
		if src[i] == end {
			if quote != '[' && i+1 < len(src) && src[i+1] == end {
				i += 2
				continue
			}
			return i + 1 - at
		}
		i++
	}
	return 0
}

// MatchPlaceholder reports how many bytes at at are a placeholder, or 0.
func MatchPlaceholder(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	if src[at] == '?' {
		return 1
	}
	if src[at] != '$' || at+1 >= len(src) || !isDigit(src[at+1]) {
		return 0
	}
	i := at + 2
	for i < len(src) && isDigit(src[i]) {
		i++
	}
	return i - at
}

// MatchNumber reports how many bytes at at are a numeric literal, or 0.
func MatchNumber(src []byte, at int) int {
	if at >= len(src) || !isDigit(src[at]) {
		return 0
	}
	i := at + 1
	for i < len(src) && isDigit(src[i]) {
		i++
	}
	if i+1 < len(src) && src[i] == '.' && isDigit(src[i+1]) {
		i++
		for i < len(src) && isDigit(src[i]) {
			i++
		}
	}
	if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
		j := i + 1
		if j < len(src) && (src[j] == '+' || src[j] == '-') {
			j++
		}
		if j < len(src) && isDigit(src[j]) {
			i = j + 1
			for i < len(src) && isDigit(src[i]) {
				i++
			}
		}
	}
	return i - at
}

// MatchOp reports how many bytes at at are one operator or punctuation mark, or 0.
func MatchOp(src string, at int) int {
	if at >= len(src) {
		return 0
	}
	if at+1 < len(src) {
		switch src[at : at+2] {
		case "<>", "!=", "<=", ">=", "||":
			return 2
		}
	}
	switch src[at] {
	case '(', ')', ',', ';', '.', '*', '=', '<', '>', '+', '-', '/', '%':
		return 1
	default:
		return 0
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// HasPrefix reports whether word occurs at at.
func HasPrefix(src string, at int, word string) bool {
	if at < 0 || at+len(word) > len(src) {
		return false
	}
	return src[at:at+len(word)] == word
}
