package dotenv

import (
	"errors"
	"fmt"
	"strings"
	"sync"

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

func define() (*xparse.Model[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]
	space := syntax.Term(TermSpace, matchSpace)
	newline := syntax.Term(TermNewline, matchNewline)
	export := syntax.Term(TermExport, matchExport)
	key := syntax.Term(TermKey, matchKey)
	equal := syntax.Term(TermEqual, matchByte('='))
	comment := syntax.Term(TermComment, matchLineComment)
	inlineComment := syntax.Term(TermInlineComment, matchHash)
	bare := syntax.Term(TermBare, matchBare)
	hash := syntax.Term(TermHash, matchValueHash)
	doubleQuote := syntax.Term(TermDoubleQuote, matchQuote('"'))
	singleQuote := syntax.Term(TermSingleQuote, matchQuote('\''))
	noted := syntax.Seq(space, inlineComment)

	tail := syntax.Eq(RuleTail, syntax.Alt(
		syntax.Seq(hash, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(space, inlineComment, syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), bare, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), doubleQuote, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), singleQuote, syntax.Opt(noted), syntax.Opt(space), syntax.Opt(newline)),
		syntax.Seq(syntax.Opt(space), syntax.Opt(newline)),
	))
	assign := syntax.Eq(RuleAssign, syntax.Seq(
		syntax.Opt(space), syntax.Opt(export), syntax.Opt(space), syntax.Opt(key), syntax.Opt(space), equal, syntax.Ref(RuleTail),
	))
	note := syntax.Eq(RuleNote, syntax.Seq(syntax.Opt(space), comment, syntax.Opt(newline)))
	blank := syntax.Eq(RuleBlank, syntax.Seq(syntax.Opt(space), newline))
	file := syntax.Eq(RuleFile, syntax.Repeat(syntax.Alt(
		syntax.Ref(RuleAssign),
		syntax.Ref(RuleNote),
		syntax.Ref(RuleBlank),
	)))
	return syntax.Compile("dotenv", syntax.Compose(file, assign, note, blank, tail, syntax.Start(RuleFile)))
}

func parse(text string) (*File, error) {
	text = strings.TrimPrefix(text, "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(text))
	if err != nil {
		var pe *xparse.Error
		if errors.As(err, &pe) {
			return nil, lineError(ErrLine, pe.Line)
		}
		return nil, lineError(ErrLine, 1)
	}
	return lower(text, root)
}

func lower(text string, root xparse.Syn[Term, Rule]) (*File, error) {
	file := New()
	var pending []string
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleBlank:
		case RuleNote:
			pending = append(pending, commentBody(textOf(child, TermComment)))
		case RuleAssign:
			assignment, err := lowerAssign(text, child)
			if err != nil {
				return nil, err
			}
			if len(pending) > 0 {
				assignment.Comment = joinComment(strings.Join(pending, "\n"), assignment.Comment)
				pending = nil
			}
			file.Assignments = append(file.Assignments, assignment)
		}
	}
	if len(pending) == 0 {
		return file, nil
	}
	note := strings.Join(pending, "\n")
	if len(file.Assignments) == 0 {
		file.Assignments = append(file.Assignments, &Assignment{Comment: note})
		return file, nil
	}
	last := file.Assignments[len(file.Assignments)-1]
	last.Comment = joinComment(last.Comment, note)
	return file, nil
}

func lowerAssign(text string, node xparse.Syn[Term, Rule]) (*Assignment, error) {
	assignment := &Assignment{Quote: QuoteBare}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleTail {
			value, quote, comment, err := lowerTail(child)
			if err != nil {
				return nil, lineError(err, lineOf(text, child.Start()))
			}
			assignment.Value = value
			assignment.Quote = quote
			assignment.Comment = comment
			continue
		}
		term, ok := child.Term()
		if !ok {
			continue
		}
		switch term {
		case TermExport:
			assignment.Export = true
		case TermKey:
			assignment.Key = child.Text()
		}
	}
	if assignment.Key == "" {
		return nil, lineError(ErrKey, lineOf(text, node.Start()))
	}
	return assignment, nil
}

func lowerTail(node xparse.Syn[Term, Rule]) (string, Quote, string, error) {
	var (
		value   string
		quote   = QuoteBare
		comment string
	)
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		term, ok := child.Term()
		if !ok {
			continue
		}
		switch term {
		case TermHash, TermBare:
			value = strings.TrimRight(child.Text(), " \t")
		case TermDoubleQuote:
			decoded, err := unescapeDouble(child.Text())
			if err != nil {
				return "", 0, "", err
			}
			value, quote = decoded, QuoteDouble
		case TermSingleQuote:
			value, quote = singleValue(child.Text()), QuoteSingle
		case TermInlineComment:
			comment = commentBody(child.Text())
		}
	}
	return value, quote, comment, nil
}

func textOf(node xparse.Syn[Term, Rule], want Term) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == want {
			return child.Text()
		}
	}
	return ""
}

func lineOf(text string, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(text) {
		off = len(text)
	}
	return strings.Count(text[:off], "\n") + 1
}

func lineError(err error, line int) error {
	if errors.Is(err, ErrKey) {
		return fmt.Errorf("%w: line %d", ErrKey, line)
	}
	return fmt.Errorf("%w: line %d", ErrLine, line)
}

func joinComment(lead, inlineComment string) string {
	if lead == "" {
		return inlineComment
	}
	if inlineComment == "" {
		return lead
	}
	return lead + "\n" + inlineComment
}

func matchSpace(src []byte, at int) int {
	i := at
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i - at
}

func matchNewline(src []byte, at int) int {
	if at < len(src) && src[at] == '\n' {
		return 1
	}
	return 0
}

func matchByte(c byte) xparse.Match {
	return func(src []byte, at int) int {
		if at < len(src) && src[at] == c {
			return 1
		}
		return 0
	}
}

// matchExport matches the keyword only when a space or tab follows it.
// The space stays for the next token, so export=1 is a key.
func matchExport(src []byte, at int) int {
	const word = "export"
	if !hasPrefix(src, at, word) {
		return 0
	}
	next := at + len(word)
	if next >= len(src) || (src[next] != ' ' && src[next] != '\t') {
		return 0
	}
	return len(word)
}

func matchKey(src []byte, at int) int {
	// A word after '=' is a bare value. Tail can be empty, so the next
	// assignment's key is in the same state as that value; a key match
	// here would steal it.
	if at >= len(src) || !keyByte(src[at], true) || afterEq(src, at) {
		return 0
	}
	i := at + 1
	for i < len(src) && keyByte(src[i], false) {
		i++
	}
	return i - at
}

func afterEq(src []byte, at int) bool {
	i := at
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	return i > 0 && src[i-1] == '='
}

func keyByte(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

// matchLineComment matches a hash only when the line so far is spaces.
func matchLineComment(src []byte, at int) int {
	if !lineHash(src, at) {
		return 0
	}
	return hashRun(src, at)
}

// matchHash matches a hash that is not a whole-line comment.
func matchHash(src []byte, at int) int {
	if lineHash(src, at) {
		return 0
	}
	return hashRun(src, at)
}

// matchValueHash matches a bare value that starts with a hash and stops
// before a spaced hash, which is an inlineComment comment.
func matchValueHash(src []byte, at int) int {
	if at >= len(src) || src[at] != '#' || lineHash(src, at) {
		return 0
	}
	return matchBareFrom(src, at)
}

func lineHash(src []byte, at int) bool {
	if at >= len(src) || src[at] != '#' {
		return false
	}
	i := at
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	return i == 0 || src[i-1] == '\n'
}

func hashRun(src []byte, at int) int {
	if at >= len(src) || src[at] != '#' {
		return 0
	}
	i := at + 1
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i - at
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

func matchQuote(quote byte) xparse.Match {
	return func(src []byte, at int) int {
		if at >= len(src) || src[at] != quote {
			return 0
		}
		i := at + 1
		for i < len(src) && src[i] != '\n' {
			if quote == '"' && src[i] == '\\' {
				if i+1 >= len(src) || src[i+1] == '\n' {
					return 0
				}
				i += 2
				continue
			}
			if src[i] == quote {
				return i + 1 - at
			}
			i++
		}
		return 0
	}
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

func hasPrefix(src []byte, at int, word string) bool {
	if at < 0 || at+len(word) > len(src) {
		return false
	}
	for i := 0; i < len(word); i++ {
		if src[at+i] != word[i] {
			return false
		}
	}
	return true
}
