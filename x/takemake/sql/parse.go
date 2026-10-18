package sql

import (
	"errors"
	"fmt"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
	syntax "github.com/lewtec/lewkit/x/parse/sql"
)

func parse(text string) (*File, error) {
	notes := scanComments(text)
	model, err := syntax.Language()
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

func syntaxAt(node xparse.SyntaxNode[syntax.Term, syntax.Rule], msg string) error {
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, node.Line(), msg)
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
			if syntax.HasPrefix(src, i, "--") {
				i += 2
				start := i
				for i < len(src) && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				addNote(&buf, src[start:i])
				continue
			}
			if syntax.HasPrefix(src, i, "/*") {
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
	if n := syntax.MatchString(buf, at); n > 0 {
		return n
	}
	if n := syntax.MatchQuoted(buf, at); n > 0 {
		return n
	}
	if n := syntax.MatchPlaceholder(buf, at); n > 0 {
		return n
	}
	if n := syntax.MatchNumber(buf, at); n > 0 {
		return n
	}
	if n := syntax.MatchIdentBytes(buf, at); n > 0 {
		return n
	}
	return syntax.MatchOp(src, at)
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
