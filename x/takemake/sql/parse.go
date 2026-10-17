package sql

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

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

func parse(text string) (*File, error) {
	notes := scanComments(text)
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(text))
	if err != nil {
		return nil, asSyntax(err)
	}
	return lowerFile(text, root, notes)
}

func asSyntax(err error) error {
	var pe *xparse.Error
	if errors.As(err, &pe) {
		line := pe.Line
		if line < 1 {
			line = 1
		}
		return fmt.Errorf("%w: line %d: %s", ErrSyntax, line, pe.Error())
	}
	return fmt.Errorf("%w: %s", ErrSyntax, err.Error())
}

func syntaxAt(src string, off int, msg string) error {
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, lineOf(src, off), msg)
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

// commentMap records the comment text that the old lexer attached to the
// token starting at each byte offset. first is that offset for the first
// token, or len(src) when the file has none.
type commentMap struct {
	first int
	saw   bool
	by    map[int]string
}

func scanComments(src string) commentMap {
	out := commentMap{by: map[int]string{}}
	var buf strings.Builder
	i := 0
	for {
		buf.Reset()
		for i < len(src) {
			switch src[i] {
			case ' ', '\t', '\n', '\r':
				i++
				continue
			}
			if hasPrefix(src, i, "--") {
				i += 2
				start := i
				for i < len(src) && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				addNote(&buf, src[start:i])
				continue
			}
			if hasPrefix(src, i, "/*") {
				end := blockEnd(src, i)
				if end < 0 {
					break
				}
				addNote(&buf, src[i+2:end])
				i = end + 2
				continue
			}
			break
		}
		if !out.saw {
			out.first = i
			out.saw = true
		}
		if text := strings.TrimSpace(buf.String()); text != "" {
			out.by[i] = text
		}
		if i >= len(src) {
			break
		}
		n := skipLexeme(src, i)
		if n <= 0 {
			break
		}
		i += n
	}
	return out
}

func addNote(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	b.WriteString(text)
}

func blockEnd(src string, at int) int {
	for i := at + 2; i+1 < len(src); i++ {
		if src[i] == '*' && src[i+1] == '/' {
			return i
		}
	}
	return -1
}

func skipLexeme(src string, at int) int {
	buf := []byte(src)
	if n := matchString(buf, at); n > 0 {
		return n
	}
	if n := matchQuoted(buf, at); n > 0 {
		return n
	}
	if n := matchPlaceholder(buf, at); n > 0 {
		return n
	}
	if n := matchNumber(buf, at); n > 0 {
		return n
	}
	if n := matchIdentBytes(buf, at); n > 0 {
		return n
	}
	return matchOp(src, at)
}

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

func matchIdentBytes(src []byte, at int) int {
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
	n := matchIdentBytes(src, at)
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

func matchSpace(src []byte, at int) int {
	i := at
	for i < len(src) {
		switch src[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			if i == at {
				return 0
			}
			return i - at
		}
	}
	if i == at {
		return 0
	}
	return i - at
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

func matchString(src []byte, at int) int {
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

func matchQuoted(src []byte, at int) int {
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

func matchPlaceholder(src []byte, at int) int {
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

func matchNumber(src []byte, at int) int {
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

func matchOp(src string, at int) int {
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

func hasPrefix(src string, at int, word string) bool {
	if at < 0 || at+len(word) > len(src) {
		return false
	}
	return src[at:at+len(word)] == word
}

func decodeString(tok string) (string, error) {
	if len(tok) < 2 || tok[0] != '\'' || tok[len(tok)-1] != '\'' {
		return "", ErrSyntax
	}
	return decodeBody(tok[1:len(tok)-1], '\''), nil
}

func decodeQuoted(tok string) (string, error) {
	if len(tok) < 2 {
		return "", ErrSyntax
	}
	end := tok[0]
	if tok[0] == '[' {
		end = ']'
	}
	if tok[len(tok)-1] != end {
		return "", ErrSyntax
	}
	body := tok[1 : len(tok)-1]
	if tok[0] == '[' {
		return decodeNewlines(body), nil
	}
	return decodeBody(body, end), nil
}

func decodeBody(body string, quote byte) string {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] == quote && i+1 < len(body) && body[i+1] == quote {
			b.WriteByte(quote)
			i++
			continue
		}
		if body[i] == '\n' || body[i] == '\r' {
			if body[i] == '\r' && i+1 < len(body) && body[i+1] == '\n' {
				i++
			}
			b.WriteByte('\n')
			continue
		}
		b.WriteByte(body[i])
	}
	return b.String()
}

func decodeNewlines(body string) string {
	return decodeBody(body, 0)
}
