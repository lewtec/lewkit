package dotenv

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
	xparse "github.com/lewtec/lewkit/x/parse"
)

// ErrLine is a dotenv line the grammar cannot read.
var ErrLine = errs.Wrap(errs.ErrSyntax, "dotenv: invalid line")

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

// Language loads the dotenv grammar.
func Language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		prepared, err := declare()
		if err != nil {
			langErr = err
			return
		}
		lang, langErr = prepared.Load(dotenvTable)
	})
	return lang, langErr
}

func declare() (*xparse.Prepared[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]
	space := syntax.Pat(TermSpace, xparse.While(" \t"))
	newline := syntax.Pat(TermNewline, xparse.Literal("\n"))
	export := syntax.Pat(TermExport, xparse.Keyword("export").Look(xparse.While(" \t")))
	key := syntax.Pat(TermKey, xparse.Class(KeyByte).Unless(xparse.After(TermEqual)))
	equal := syntax.Pat(TermEqual, xparse.Byte('='))
	comment := syntax.Pat(TermComment, xparse.Line("#").When(xparse.AtLine()))
	inlineComment := syntax.Pat(TermInlineComment, xparse.Line("#").Unless(xparse.AtLine()))
	bare := syntax.Pat(TermBare, xparse.Func(matchBare))
	hash := syntax.Pat(TermHash, xparse.Func(matchValueHash))
	doubleQuote := syntax.Pat(TermDoubleQuote, xparse.Quote('"'))
	singleQuote := syntax.Pat(TermSingleQuote, xparse.Quote('\'').Raw())
	noted := syntax.Seq(space, inlineComment)

	tail := syntax.Eq(RuleTail, syntax.Alt(
		syntax.Seq(hash, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(space, inlineComment, syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), bare, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), doubleQuote, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), singleQuote, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), syntax.Opt(newline)),
	)).Build(decodeTail)
	assign := syntax.Eq(RuleAssign, syntax.Seq(
		syntax.Skip(syntax.Opt(space)),
		syntax.Flag(syntax.Opt(export), "Export"),
		syntax.Skip(syntax.Opt(space)),
		syntax.Text(syntax.Opt(key), "Key"),
		syntax.Skip(syntax.Opt(space)),
		syntax.Skip(equal),
		syntax.Text(syntax.Ref(RuleTail), "Value"),
	))
	note := syntax.Eq(RuleNote, syntax.Seq(
		syntax.Skip(syntax.Opt(space)),
		syntax.Text(comment, "Comment"),
		syntax.Skip(syntax.Opt(newline)),
	)).Build(decodeNote)
	blank := syntax.Eq(RuleBlank, syntax.Seq(syntax.Skip(syntax.Opt(space)), newline))
	file := syntax.Eq(RuleFile, syntax.Repeat(syntax.Alt(
		syntax.Ref(RuleAssign),
		syntax.Ref(RuleNote),
		syntax.Ref(RuleBlank),
	)))
	return syntax.Prepare("dotenv", syntax.Compose(
		file, assign, note, blank, tail, syntax.Start(RuleFile),
	))
}

func decodeNote(node xparse.SyntaxNode[Term, Rule]) (any, error) {
	var line struct {
		Comment string
	}
	if err := node.Fill(&line); err != nil {
		return nil, err
	}
	return commentBody(line.Comment), nil
}

type tailValue struct {
	Value   string
	Quote   Quote
	Comment string
}

func decodeTail(node xparse.SyntaxNode[Term, Rule]) (any, error) {
	decoded := tailValue{Quote: QuoteBare}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		term, ok := child.Term()
		if !ok {
			continue
		}
		switch term {
		case TermHash, TermBare:
			decoded.Value = strings.TrimRight(child.Text(), " \t")
		case TermDoubleQuote:
			value, err := unescapeDouble(child.Text())
			if err != nil {
				return nil, fmt.Errorf("%w: line %d", ErrLine, node.Line())
			}
			decoded.Value = value
			decoded.Quote = QuoteDouble
		case TermSingleQuote:
			decoded.Value = singleValue(child.Text())
			decoded.Quote = QuoteSingle
		case TermInlineComment:
			decoded.Comment = commentBody(child.Text())
		}
	}
	return decoded, nil
}

// KeyByte reports whether c may appear in a dotenv key. first is the first byte.
func KeyByte(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

// matchValueHash matches a bare value that starts with a hash and stops
// before a spaced hash, which is an inline comment.
func matchValueHash(src []byte, at int) int {
	if at >= len(src) || src[at] != '#' || atLineStart(src, at) {
		return 0
	}
	return matchBareFrom(src, at)
}

func atLineStart(src []byte, at int) bool {
	if at >= len(src) || src[at] != '#' {
		return false
	}
	i := at
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	return i == 0 || src[i-1] == '\n'
}

func matchBare(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	switch src[at] {
	case ' ', '\t', '\n', '#', '"', '\'', '=':
		return 0
	}
	return matchBareFrom(src, at)
}

func matchBareFrom(src []byte, at int) int {
	i := at + 1
	for i < len(src) && src[i] != '\n' {
		if src[i] == '#' && (src[i-1] == ' ' || src[i-1] == '\t') {
			break
		}
		i++
	}
	end := i
	for end > at && (src[end-1] == ' ' || src[end-1] == '\t') {
		end--
	}
	if end <= at {
		return 0
	}
	return end - at
}

func unescapeDouble(tok string) (string, error) {
	if len(tok) < 2 || tok[0] != '"' || tok[len(tok)-1] != '"' {
		return "", ErrLine
	}
	s := tok[1 : len(tok)-1]
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			return "", ErrLine
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String(), nil
}

func singleValue(tok string) string {
	if len(tok) >= 2 && tok[0] == '\'' && tok[len(tok)-1] == '\'' {
		return tok[1 : len(tok)-1]
	}
	return tok
}

func commentBody(s string) string {
	s = strings.TrimPrefix(s, "#")
	return strings.TrimPrefix(s, " ")
}
