package parse

// Pattern is the shape of one token. [Match] is the compiled form.
// Constructors return a new pattern and do not mutate an existing one.
type Pattern struct {
	kind    patKind
	text    string
	test    func(byte, bool) bool
	inner   *Pattern
	next    *Pattern
	cond    Condition
	match   Match
	lex     int
	hasLex  bool
	unit    byte
	escapes bool
}

type patKind uint8

const (
	patWhile patKind = iota + 1
	patLiteral
	patKeyword
	patClass
	patLine
	patQuote
	patByte
	patFunc
	patUnless
	patWhen
	patLook
)

// Condition is a reason for [Pattern.Unless] or [Pattern.When] to reject a match.
type Condition struct {
	kind condKind
	name string
}

type condKind uint8

const (
	condAfter condKind = iota + 1
	condLine
)

// While matches one or more bytes from chars.
func While(chars string) Pattern {
	return Class(func(c byte, _ bool) bool {
		for i := 0; i < len(chars); i++ {
			if chars[i] == c {
				return true
			}
		}
		return false
	})
}

// Literal matches text exactly. Its lexical precedence is 1.
func Literal(text string) Pattern {
	return Pattern{kind: patLiteral, text: text, lex: 1, hasLex: true}
}

// Keyword matches text when the next byte is not an ASCII word byte.
// Its lexical precedence is 2, the same band as [Grammar.Keyword].
func Keyword(text string) Pattern {
	return Pattern{kind: patKeyword, text: text, lex: 2, hasLex: true}
}

// Class matches one or more bytes. test reports whether c may appear.
// first is true only for the byte at the start of the token.
func Class(test func(c byte, first bool) bool) Pattern {
	return Pattern{kind: patClass, test: test}
}

// Line matches mark and the rest of the line, not including the newline.
func Line(mark string) Pattern {
	return Pattern{kind: patLine, text: mark}
}

// Quote matches a quoted span. A backslash escapes the next byte.
// A newline ends the match as a failure. [Pattern.Raw] turns escapes off.
func Quote(quote byte) Pattern {
	return Pattern{kind: patQuote, unit: quote, escapes: true}
}

// Byte matches one byte. Its lexical precedence is 1.
func Byte(c byte) Pattern {
	return Pattern{kind: patByte, unit: c, lex: 1, hasLex: true}
}

// Func is a matcher the constructors do not cover.
func Func(match Match) Pattern {
	return Pattern{kind: patFunc, match: match}
}

// Unless rejects the pattern when cond holds.
func (p Pattern) Unless(cond Condition) Pattern {
	cp := p
	return Pattern{kind: patUnless, inner: &cp, cond: cond, lex: p.lex, hasLex: p.hasLex}
}

// When keeps the pattern only when cond holds.
func (p Pattern) When(cond Condition) Pattern {
	cp := p
	return Pattern{kind: patWhen, inner: &cp, cond: cond, lex: p.lex, hasLex: p.hasLex}
}

// Look keeps the pattern only when next matches at the end. next is not consumed.
func (p Pattern) Look(next Pattern) Pattern {
	cp := p
	n := next
	return Pattern{kind: patLook, inner: &cp, next: &n, lex: p.lex, hasLex: p.hasLex}
}

// Raw turns escapes off on a [Quote] pattern.
func (p Pattern) Raw() Pattern {
	p.escapes = false
	return p
}

// After is the previous terminal. A token of only spaces or tabs is not previous.
func After(term interface{ String() string }) Condition {
	if term == nil {
		return Condition{}
	}
	return Condition{kind: condAfter, name: term.String()}
}

// AtLine holds when only spaces or tabs stand between this byte and the
// start of the line.
func AtLine() Condition {
	return Condition{kind: condLine}
}

type builtPat struct {
	match Match
	prev  func(src []byte, at int, prev SymbolID) int
}

func (b builtPat) eval(src []byte, at int, prev SymbolID) int {
	if b.prev != nil {
		return b.prev(src, at, prev)
	}
	if b.match == nil {
		return 0
	}
	return b.match(src, at)
}

func (p Pattern) build(name func(SymbolID) string) builtPat {
	switch p.kind {
	case patWhile, patClass:
		return builtPat{match: classMatch(p.test)}
	case patLiteral:
		return builtPat{match: hasBytes(p.text)}
	case patKeyword:
		return builtPat{match: keywordMatch(p.text)}
	case patLine:
		return builtPat{match: lineMatch(p.text)}
	case patQuote:
		return builtPat{match: quoteMatch(p.unit, p.escapes)}
	case patByte:
		return builtPat{match: byteMatch(p.unit)}
	case patFunc:
		return builtPat{match: p.match}
	case patUnless, patWhen:
		if p.inner == nil {
			return builtPat{}
		}
		inner := p.inner.build(name)
		cond := p.cond
		keep := p.kind == patWhen
		return builtPat{prev: func(src []byte, at int, prev SymbolID) int {
			holds := cond.holds(src, at, prev, name)
			if keep != holds {
				return 0
			}
			return inner.eval(src, at, prev)
		}}
	case patLook:
		if p.inner == nil || p.next == nil {
			return builtPat{}
		}
		inner := p.inner.build(name)
		ahead := p.next.build(name)
		if inner.prev == nil && ahead.prev == nil {
			return builtPat{match: func(src []byte, at int) int {
				n := inner.eval(src, at, 0)
				if n <= 0 || ahead.eval(src, at+n, 0) <= 0 {
					return 0
				}
				return n
			}}
		}
		return builtPat{prev: func(src []byte, at int, prev SymbolID) int {
			n := inner.eval(src, at, prev)
			if n <= 0 || ahead.eval(src, at+n, prev) <= 0 {
				return 0
			}
			return n
		}}
	default:
		return builtPat{}
	}
}

func (c Condition) holds(src []byte, at int, prev SymbolID, name func(SymbolID) string) bool {
	switch c.kind {
	case condAfter:
		if name == nil || c.name == "" {
			return false
		}
		return name(prev) == c.name
	case condLine:
		return atLine(src, at)
	default:
		return false
	}
}

func atLine(src []byte, at int) bool {
	i := at
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	return i == 0 || src[i-1] == '\n'
}

func classMatch(test func(byte, bool) bool) Match {
	return func(src []byte, at int) int {
		if test == nil || at >= len(src) || !test(src[at], true) {
			return 0
		}
		i := at + 1
		for i < len(src) && test(src[i], false) {
			i++
		}
		return i - at
	}
}

func keywordMatch(text string) Match {
	return func(src []byte, at int) int {
		if text == "" || !hasPrefix(src, at, text) {
			return 0
		}
		end := at + len(text)
		if end < len(src) && wordByte(src[end]) {
			return 0
		}
		return len(text)
	}
}

func wordByte(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func lineMatch(mark string) Match {
	return func(src []byte, at int) int {
		if mark == "" || !hasPrefix(src, at, mark) {
			return 0
		}
		i := at + len(mark)
		for i < len(src) && src[i] != '\n' {
			i++
		}
		return i - at
	}
}

func quoteMatch(quote byte, escapes bool) Match {
	return func(src []byte, at int) int {
		if at >= len(src) || src[at] != quote {
			return 0
		}
		i := at + 1
		for i < len(src) && src[i] != '\n' {
			if escapes && src[i] == '\\' {
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

func byteMatch(c byte) Match {
	return func(src []byte, at int) int {
		if at < len(src) && src[at] == c {
			return 1
		}
		return 0
	}
}
