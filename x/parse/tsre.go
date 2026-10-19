package parse

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/errs"
)

// rex is a token pattern from a tree-sitter grammar. Alternation is
// leftmost-first. Repetition is greedy and gives up characters when the
// rest of the pattern needs them.
type rex struct {
	op   byte
	lit  string
	cls  *rexClass
	sub  []*rex
	min  int
	max  int // negative means no upper bound
	prop func(rune) bool
}

const (
	opEmpty byte = iota
	opLit
	opClass
	opAny
	opCat
	opAlt
	opRep
	opProp
	opEnd
)

type rexClass struct {
	neg   bool
	runes []rune // singles and pairs (lo, hi) with hi bit? store ranges as [2]rune
	spans [][2]rune
	props []func(rune) bool
	nots  []func(rune) bool
}

func (c *rexClass) has(r rune) bool {
	in := false
	for _, s := range c.runes {
		if r == s {
			in = true
			break
		}
	}
	if !in {
		for _, sp := range c.spans {
			if r >= sp[0] && r <= sp[1] {
				in = true
				break
			}
		}
	}
	if !in {
		for _, p := range c.props {
			if p(r) {
				in = true
				break
			}
		}
	}
	for _, p := range c.nots {
		if p(r) {
			in = false
			break
		}
	}
	if c.neg {
		return !in
	}
	return in
}

func reEmpty() *rex { return &rex{op: opEmpty} }

func reLit(s string) *rex {
	if s == "" {
		return reEmpty()
	}
	return &rex{op: opLit, lit: s}
}

func reCat(xs ...*rex) *rex {
	var flat []*rex
	for _, x := range xs {
		if x == nil || x.op == opEmpty {
			continue
		}
		if x.op == opCat {
			flat = append(flat, x.sub...)
			continue
		}
		flat = append(flat, x)
	}
	if len(flat) == 0 {
		return reEmpty()
	}
	if len(flat) == 1 {
		return flat[0]
	}
	return &rex{op: opCat, sub: flat}
}

func reAlt(xs ...*rex) *rex {
	if len(xs) == 0 {
		return reEmpty()
	}
	if len(xs) == 1 {
		return xs[0]
	}
	return &rex{op: opAlt, sub: xs}
}

func reRep(x *rex, min, max int) *rex {
	if x == nil || x.op == opEmpty {
		return reEmpty()
	}
	return &rex{op: opRep, sub: []*rex{x}, min: min, max: max}
}

// matchLen reports how many bytes the pattern consumes at at.
// Zero means it does not match. Fuel bounds backtracking.
func (e *rex) matchLen(src []byte, at int) int {
	if e == nil {
		return 0
	}
	fuel := 200000
	end := -1
	e.match(src, at, &fuel, func(j int) bool {
		end = j
		return true
	})
	if end <= at {
		return 0
	}
	return end - at
}

func (e *rex) match(src []byte, i int, fuel *int, then func(int) bool) bool {
	if *fuel <= 0 {
		return false
	}
	*fuel--
	switch e.op {
	case opEmpty:
		return then(i)
	case opLit:
		if i+len(e.lit) <= len(src) && string(src[i:i+len(e.lit)]) == e.lit {
			return then(i + len(e.lit))
		}
		return false
	case opClass:
		if i >= len(src) {
			return false
		}
		r, n := utf8.DecodeRune(src[i:])
		if r == utf8.RuneError && n == 1 {
			return false
		}
		if e.cls.has(r) {
			return then(i + n)
		}
		return false
	case opProp:
		if i >= len(src) {
			return false
		}
		r, n := utf8.DecodeRune(src[i:])
		if r == utf8.RuneError && n == 1 || !e.prop(r) {
			return false
		}
		return then(i + n)
	case opAny:
		if i >= len(src) {
			return false
		}
		r, n := utf8.DecodeRune(src[i:])
		if r == utf8.RuneError && n == 1 || r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029' {
			return false
		}
		return then(i + n)
	case opCat:
		var rec func(k, i int) bool
		rec = func(k, i int) bool {
			if k == len(e.sub) {
				return then(i)
			}
			return e.sub[k].match(src, i, fuel, func(j int) bool { return rec(k+1, j) })
		}
		return rec(0, i)
	case opAlt:
		for _, s := range e.sub {
			if s.match(src, i, fuel, then) {
				return true
			}
		}
		return false
	case opEnd:
		if i == len(src) {
			return then(i)
		}
		return false
	case opRep:
		body := e.sub[0]
		var rec func(taken, i int) bool
		rec = func(taken, i int) bool {
			if e.max < 0 || taken < e.max {
				if body.match(src, i, fuel, func(j int) bool {
					if j == i {
						return false
					}
					return rec(taken+1, j)
				}) {
					return true
				}
			}
			if taken >= e.min {
				return then(i)
			}
			return false
		}
		return rec(0, i)
	default:
		return false
	}
}

type reParser struct {
	s string
	i int
}

func patternSyntax(format string, args ...any) error {
	return errs.Wrap(errs.ErrSyntax, fmt.Sprintf(format, args...))
}

func compilePattern(pat string) (*rex, error) {
	p := &reParser{s: pat}
	re, err := p.alt()
	if err != nil {
		return nil, err
	}
	if p.i != len(p.s) {
		return nil, patternSyntax("pattern %q stopped at %q", pat, p.s[p.i:])
	}
	return re, nil
}

func (p *reParser) alt() (*rex, error) {
	left, err := p.seq()
	if err != nil {
		return nil, err
	}
	xs := []*rex{left}
	for p.peek('|') {
		p.i++
		right, err := p.seq()
		if err != nil {
			return nil, err
		}
		xs = append(xs, right)
	}
	return reAlt(xs...), nil
}

func (p *reParser) seq() (*rex, error) {
	var xs []*rex
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case '|', ')':
			return reCat(xs...), nil
		}
		a, err := p.atom()
		if err != nil {
			return nil, err
		}
		xs = append(xs, a)
	}
	return reCat(xs...), nil
}

func (p *reParser) atom() (*rex, error) {
	if p.i >= len(p.s) {
		return nil, patternSyntax("pattern ended early")
	}
	var x *rex
	var err error
	switch p.s[p.i] {
	case '(':
		p.i++
		if p.i+1 < len(p.s) && p.s[p.i] == '?' {
			if p.s[p.i+1] != ':' {
				return nil, patternSyntax("unsupported group %q", p.s[p.i:])
			}
			p.i += 2
		}
		x, err = p.alt()
		if err != nil {
			return nil, err
		}
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil, patternSyntax("unclosed group")
		}
		p.i++
	case '[':
		x, err = p.class()
		if err != nil {
			return nil, err
		}
	case '.':
		p.i++
		x = &rex{op: opAny}
	case '\\':
		x, err = p.escape(false)
		if err != nil {
			return nil, err
		}
	case '^':
		// These grammars do not use a start anchor outside a class.
		p.i++
		x = reEmpty()
	case '$':
		p.i++
		x = &rex{op: opEnd}
	default:
		if p.s[p.i] == '*' || p.s[p.i] == '+' || p.s[p.i] == '?' || p.s[p.i] == '{' {
			return nil, patternSyntax("quantifier without an atom at %q", p.s[p.i:])
		}
		x = reLit(p.s[p.i : p.i+1])
		p.i++
	}
	return p.quant(x)
}

func (p *reParser) quant(x *rex) (*rex, error) {
	if p.i >= len(p.s) {
		return x, nil
	}
	min, max := 0, 0
	switch p.s[p.i] {
	case '*':
		min, max = 0, -1
	case '+':
		min, max = 1, -1
	case '?':
		min, max = 0, 1
	case '{':
		var err error
		min, max, err = p.braces()
		if err != nil {
			return nil, err
		}
	default:
		return x, nil
	}
	p.i++
	if p.i < len(p.s) && p.s[p.i] == '?' {
		// Lazy quantifiers are taken as greedy. Token patterns in these
		// grammars do not use them outside character classes.
		p.i++
	}
	return reRep(x, min, max), nil
}

func (p *reParser) braces() (int, int, error) {
	p.i++ // skip {
	n, ok := p.num()
	if !ok {
		return 0, 0, patternSyntax("bad brace quantifier")
	}
	if p.i < len(p.s) && p.s[p.i] == '}' {
		return n, n, nil
	}
	if p.i >= len(p.s) || p.s[p.i] != ',' {
		return 0, 0, patternSyntax("bad brace quantifier")
	}
	p.i++
	if p.i < len(p.s) && p.s[p.i] == '}' {
		return n, -1, nil
	}
	m, ok := p.num()
	if !ok || p.i >= len(p.s) || p.s[p.i] != '}' {
		return 0, 0, patternSyntax("bad brace quantifier")
	}
	return n, m, nil
}

func (p *reParser) num() (int, bool) {
	if p.i >= len(p.s) || p.s[p.i] < '0' || p.s[p.i] > '9' {
		return 0, false
	}
	n := 0
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		n = n*10 + int(p.s[p.i]-'0')
		p.i++
	}
	return n, true
}

func (p *reParser) class() (*rex, error) {
	p.i++ // [
	c := &rexClass{}
	if p.peek('^') {
		c.neg = true
		p.i++
	}
	first := true
	for {
		if p.i >= len(p.s) {
			return nil, patternSyntax("unclosed class")
		}
		if p.s[p.i] == ']' && !first {
			p.i++
			break
		}
		first = false
		lo, prop, negProp, err := p.classAtom()
		if err != nil {
			return nil, err
		}
		if prop != nil {
			if negProp {
				c.nots = append(c.nots, prop)
			} else {
				c.props = append(c.props, prop)
			}
			continue
		}
		if p.i+1 < len(p.s) && p.s[p.i] == '-' && p.s[p.i+1] != ']' {
			p.i++
			hi, p2, _, err := p.classAtom()
			if err != nil {
				return nil, err
			}
			if p2 != nil {
				return nil, patternSyntax("property in a class range")
			}
			if hi < lo {
				return nil, patternSyntax("backwards class range")
			}
			c.spans = append(c.spans, [2]rune{lo, hi})
			continue
		}
		c.runes = append(c.runes, lo)
	}
	return &rex{op: opClass, cls: c}, nil
}

func (p *reParser) classAtom() (rune, func(rune) bool, bool, error) {
	if p.i >= len(p.s) {
		return 0, nil, false, patternSyntax("class ended early")
	}
	if p.s[p.i] != '\\' {
		r, n := utf8.DecodeRuneInString(p.s[p.i:])
		p.i += n
		return r, nil, false, nil
	}
	x, err := p.escape(true)
	if err != nil {
		return 0, nil, false, err
	}
	if x.op == opProp {
		return 0, x.prop, false, nil
	}
	if x.op == opClass && x.cls != nil && len(x.cls.props) == 1 && !x.cls.neg {
		return 0, x.cls.props[0], false, nil
	}
	if x.op == opClass && x.cls != nil && x.cls.neg && len(x.cls.props) == 1 {
		return 0, x.cls.props[0], true, nil
	}
	if x.op == opLit && x.lit != "" {
		r, _ := utf8.DecodeRuneInString(x.lit)
		return r, nil, false, nil
	}
	return 0, nil, false, patternSyntax("escape cannot sit in a class")
}

func (p *reParser) escape(inClass bool) (*rex, error) {
	if p.i >= len(p.s) || p.s[p.i] != '\\' {
		return nil, patternSyntax("expected escape")
	}
	p.i++
	if p.i >= len(p.s) {
		return nil, patternSyntax("trailing backslash")
	}
	c := p.s[p.i]
	p.i++
	switch c {
	case 'n':
		return reLit("\n"), nil
	case 'r':
		return reLit("\r"), nil
	case 't':
		return reLit("\t"), nil
	case 'f':
		return reLit("\f"), nil
	case 'v':
		return reLit("\v"), nil
	case '0':
		return reLit("\x00"), nil
	case 'd':
		return classFn(isDigit, false), nil
	case 'D':
		return classFn(isDigit, true), nil
	case 's':
		return classFn(unicode.IsSpace, false), nil
	case 'S':
		return classFn(unicode.IsSpace, true), nil
	case 'w':
		return classFn(isWord, false), nil
	case 'W':
		return classFn(isWord, true), nil
	case 'p', 'P':
		if p.i >= len(p.s) || p.s[p.i] != '{' {
			return nil, patternSyntax("property escape wants a name")
		}
		p.i++
		start := p.i
		for p.i < len(p.s) && p.s[p.i] != '}' {
			p.i++
		}
		if p.i >= len(p.s) {
			return nil, patternSyntax("unclosed property")
		}
		name := p.s[start:p.i]
		p.i++
		fn := unicodeProp(name)
		if fn == nil {
			return nil, patternSyntax("unknown property %s", name)
		}
		if c == 'P' {
			return classFn(fn, true), nil
		}
		return &rex{op: opProp, prop: fn}, nil
	case 'x':
		r, err := p.hex(2)
		if err != nil {
			return nil, err
		}
		return reLit(string(r)), nil
	case 'u':
		if p.i < len(p.s) && p.s[p.i] == '{' {
			p.i++
			start := p.i
			for p.i < len(p.s) && p.s[p.i] != '}' {
				p.i++
			}
			if p.i >= len(p.s) {
				return nil, patternSyntax("unclosed hex rune")
			}
			h := p.s[start:p.i]
			p.i++
			r, err := parseHexRune(h)
			if err != nil {
				return nil, err
			}
			return reLit(string(r)), nil
		}
		r, err := p.hex(4)
		if err != nil {
			return nil, err
		}
		return reLit(string(r)), nil
	case 'b':
		if inClass {
			return reLit("\b"), nil
		}
		return nil, patternSyntax("word boundary is not a token pattern")
	default:
		return reLit(string(c)), nil
	}
}

func (p *reParser) hex(n int) (rune, error) {
	if p.i+n > len(p.s) {
		return 0, patternSyntax("short hex escape")
	}
	r, err := parseHexRune(p.s[p.i : p.i+n])
	p.i += n
	return r, err
}

func parseHexRune(h string) (rune, error) {
	var v rune
	if h == "" {
		return 0, patternSyntax("empty hex escape")
	}
	for _, c := range h {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v += c - '0'
		case c >= 'a' && c <= 'f':
			v += c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v += c - 'A' + 10
		default:
			return 0, patternSyntax("bad hex escape %q", h)
		}
	}
	return v, nil
}

func (p *reParser) peek(c byte) bool {
	return p.i < len(p.s) && p.s[p.i] == c
}

func classFn(fn func(rune) bool, neg bool) *rex {
	c := &rexClass{neg: neg, props: []func(rune) bool{fn}}
	return &rex{op: opClass, cls: c}
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isWord(r rune) bool {
	return isDigit(r) || r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func unicodeProp(name string) func(rune) bool {
	switch name {
	case "XID_Start":
		return xidStart
	case "XID_Continue":
		return xidContinue
	}
	if t, ok := unicode.Categories[name]; ok {
		return func(r rune) bool { return unicode.Is(t, r) }
	}
	if t, ok := unicode.Properties[name]; ok {
		return func(r rune) bool { return unicode.Is(t, r) }
	}
	if t, ok := unicode.Scripts[name]; ok {
		return func(r rune) bool { return unicode.Is(t, r) }
	}
	return nil
}

func xidStart(r rune) bool {
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) || unicode.Is(unicode.Other_ID_Start, r)
}

func xidContinue(r rune) bool {
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return xidStart(r) || unicode.In(r, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc) || unicode.Is(unicode.Other_ID_Continue, r)
}
