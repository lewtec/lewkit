package parse

import (
	"fmt"
	"unicode/utf8"
)

// SymbolID is a terminal or a rule in one compiled language.
type SymbolID uint16

const (
	// SymEOF is the end of the source. Compile reserves it.
	SymEOF SymbolID = 1
)

// Assoc is how a production breaks a precedence tie.
type Assoc uint8

const (
	// AssocNone keeps both actions when the precedences are equal.
	AssocNone Assoc = iota
	// AssocLeft reduces on a tie.
	AssocLeft
	// AssocRight shifts on a tie.
	AssocRight
)

// Match reports how many bytes at at belong to a token, or 0.
type Match func(src []byte, at int) int

// ScanFunc reads one token. valid reports whether the current state has an
// action on that symbol. A scanner that owns the whole token stream may
// ignore valid. st is reused for the whole file and starts zeroed, except
// Symbol, which resolves a terminal name.
type ScanFunc func(src []byte, at int, valid func(SymbolID) bool, st *LexState) (id SymbolID, start, end int, ok bool)

// LexState is the scanner's memory between tokens.
// Symbol is set by the parser. The other fields belong to the scanner.
type LexState struct {
	Symbol func(name string) (SymbolID, bool)
	// At is the offset the parser will pass on the next call. A scanner
	// that keeps its own cursor may leave At alone; the parser then uses
	// the token end. A scanner may also set At when the next byte is not
	// the token end.
	At int
	// InsertSemi is the Go rule: the previous token wants a semicolon
	// before a newline or the end of the file.
	InsertSemi bool
	// SemiAt is the offset of a semicolon the scanner already decided to
	// emit, or -1. Hold is set when that semicolon follows a block comment.
	SemiAt int
	Hold   bool
	// Data is private scanner state for one Parse call.
	Data any
}

// Grammar is the model Compile reads. [Build] places a [Form] into one.
type Grammar struct {
	name     string
	syms     []symRec
	prods    []prodRec
	byName   map[string]SymbolID
	extras   []SymbolID
	scan     ScanFunc
	start    SymbolID
	startSet bool
}

type kind uint8

const (
	kindNonterm kind = iota
	kindTerm
)

type symRec struct {
	name      string
	kind      kind
	match     Match
	cont      func(rune) bool
	lexPrec   int
	named     bool
	hidden    bool
	extra     bool
	immediate bool
	// external is a tree-sitter scanner token. It has no matcher.
	external bool
}

type prodRec struct {
	lhs     SymbolID
	rhs     []SymbolID
	prec    int
	hasPrec bool
	assoc   Assoc
	dyn     int
	hasDyn  bool
}

// Symbol is one token or rule. Methods on a token fill in how it matches.
type Symbol struct {
	g  *Grammar
	id SymbolID
}

// Rule is a nonterminal. Each Seq adds one alternative.
type Rule struct {
	g  *Grammar
	id SymbolID
}

// Prod is one alternative. Precedence methods record how Compile should
// break a shift/reduce conflict.
type Prod struct {
	g *Grammar
	i int
}

// New starts a grammar. name is the language name carried on the compiled value.
func New(name string) *Grammar {
	g := &Grammar{
		name:   name,
		byName: map[string]SymbolID{},
		// id 0 is unused so a zero SymbolID is never a real symbol.
		// id 1 is EOF.
		syms: []symRec{{}, {name: "eof", kind: kindTerm, named: false}},
	}
	return g
}

// Scanner installs a whole-file scanner. The default lexer is not used.
func (g *Grammar) Scanner(fn ScanFunc) *Grammar {
	g.scan = fn
	return g
}

// Start selects the root rule. Without it, the first Rule is the root.
func (g *Grammar) Start(name string) *Grammar {
	s := g.Sym(name)
	g.start = s.id
	g.startSet = true
	return g
}

// Skip marks a token as trivia. The default lexer consumes it wherever it
// matches and does not put it in the tree. A scanner-based language skips
// its own trivia and does not need Skip.
func (g *Grammar) Skip(s *Symbol) *Grammar {
	if s == nil || s.g != g {
		return g
	}
	g.syms[s.id].extra = true
	g.syms[s.id].kind = kindTerm
	g.extras = append(g.extras, s.id)
	return g
}

// Token declares a named terminal. A later Word, Match, or Literal fills
// the matcher. A scanner-based language may leave the matcher empty.
func (g *Grammar) Token(name string) *Symbol {
	return g.term(name, true)
}

// Literal declares an anonymous exact-text token. The same text is one symbol.
func (g *Grammar) Literal(text string) *Symbol {
	if id, ok := g.byName[text]; ok {
		return &Symbol{g: g, id: id}
	}
	s := g.term(text, false)
	g.syms[s.id].match = hasBytes(text)
	g.syms[s.id].lexPrec = 1
	return s
}

// Keyword declares a word that is that keyword only when the state accepts
// it. Otherwise the word symbol matches the same bytes. word is the identifier
// token whose continuation test protects the boundary, so "if" does not match
// inside "iff".
func (g *Grammar) Keyword(text string, word *Symbol) *Symbol {
	s := g.Literal(text)
	cont := func(rune) bool { return false }
	if word != nil && word.g == g && g.syms[word.id].cont != nil {
		cont = g.syms[word.id].cont
	}
	g.syms[s.id].match = func(src []byte, at int) int {
		if !hasPrefix(src, at, text) {
			return 0
		}
		end := at + len(text)
		if end < len(src) {
			r, _ := utf8.DecodeRune(src[end:])
			if cont(r) {
				return 0
			}
		}
		return len(text)
	}
	g.syms[s.id].lexPrec = 2
	return s
}

// Sym returns the symbol name, creating a rule if it has not been declared.
func (g *Grammar) Sym(name string) *Symbol {
	if id, ok := g.byName[name]; ok {
		return &Symbol{g: g, id: id}
	}
	return g.add(name, kindNonterm, false)
}

// Rule returns the rule name, creating it if needed.
func (g *Grammar) Rule(name string) *Rule {
	s := g.Sym(name)
	if g.syms[s.id].kind == kindTerm {
		// A token and a rule cannot share a name. Compile reports it.
		return &Rule{g: g, id: s.id}
	}
	g.syms[s.id].kind = kindNonterm
	if isHidden(name) {
		g.syms[s.id].hidden = true
		g.syms[s.id].named = false
	} else {
		g.syms[s.id].named = true
	}
	return &Rule{g: g, id: s.id}
}

// Repeat is zero or more. The rule is hidden and flattened into its parent.
func (g *Grammar) Repeat(s *Symbol) *Symbol {
	r := g.hidden("$repeat")
	r.Seq()
	r.Seq(g.Sym(g.syms[r.id].name), s)
	return g.Sym(g.syms[r.id].name)
}

// Repeat1 is one or more.
func (g *Grammar) Repeat1(s *Symbol) *Symbol {
	return g.hidden("$repeat1").Seq(s, g.Repeat(s)).symbol()
}

// Opt is the symbol or nothing. The rule is hidden.
func (g *Grammar) Opt(s *Symbol) *Symbol {
	r := g.hidden("$opt")
	r.Seq()
	r.Seq(s)
	return g.Sym(g.syms[r.id].name)
}

// Sep is zero or more items separated by sep.
func (g *Grammar) Sep(item, sep *Symbol) *Symbol {
	return g.Opt(g.Sep1(item, sep))
}

// Sep1 is one or more items separated by sep.
func (g *Grammar) Sep1(item, sep *Symbol) *Symbol {
	return g.hidden("$sep").Seq(item, g.Repeat(g.hidden("$sepp").Seq(sep, item).symbol())).symbol()
}

// Seq appends one alternative.
func (r *Rule) Seq(parts ...*Symbol) *Prod {
	rhs := make([]SymbolID, len(parts))
	for i, p := range parts {
		if p == nil || p.g != r.g {
			continue
		}
		rhs[i] = p.id
	}
	r.g.prods = append(r.g.prods, prodRec{lhs: r.id, rhs: rhs})
	return &Prod{g: r.g, i: len(r.g.prods) - 1}
}

func (p *Prod) symbol() *Symbol {
	return &Symbol{g: p.g, id: p.g.prods[p.i].lhs}
}

// Left sets this alternative's precedence and left associativity.
// Compile reads that precedence from the item that shifts or reduces,
// so the same token can shift at one precedence and reduce at another.
func (p *Prod) Left(n int) *Prod {
	p.g.prods[p.i].prec = n
	p.g.prods[p.i].hasPrec = true
	p.g.prods[p.i].assoc = AssocLeft
	return p
}

// Right sets precedence and right associativity.
func (p *Prod) Right(n int) *Prod {
	p.g.prods[p.i].prec = n
	p.g.prods[p.i].hasPrec = true
	p.g.prods[p.i].assoc = AssocRight
	return p
}

// Prec sets precedence and leaves a tie as a fork.
func (p *Prod) Prec(n int) *Prod {
	p.g.prods[p.i].prec = n
	p.g.prods[p.i].hasPrec = true
	p.g.prods[p.i].assoc = AssocNone
	return p
}

// Dynamic sets the runtime precedence used when two forks cover the same span.
// The node stores this value plus the dynamic precedence of its children.
func (p *Prod) Dynamic(n int) *Prod {
	p.g.prods[p.i].dyn = n
	p.g.prods[p.i].hasDyn = true
	return p
}

// Word matches an identifier. start and cont classify runes.
func (s *Symbol) Word(start, cont func(rune) bool) *Symbol {
	s.g.syms[s.id].kind = kindTerm
	s.g.syms[s.id].cont = cont
	s.g.syms[s.id].match = func(src []byte, at int) int {
		if at >= len(src) {
			return 0
		}
		r, n := utf8.DecodeRune(src[at:])
		if r == utf8.RuneError && n == 1 || !start(r) {
			return 0
		}
		i := at + n
		for i < len(src) {
			r, n = utf8.DecodeRune(src[i:])
			if r == utf8.RuneError && n == 1 || !cont(r) {
				break
			}
			i += n
		}
		return i - at
	}
	return s
}

// Match sets the token's matcher.
func (s *Symbol) Match(fn Match) *Symbol {
	s.g.syms[s.id].kind = kindTerm
	s.g.syms[s.id].match = fn
	return s
}

func (g *Grammar) term(name string, named bool) *Symbol {
	if id, ok := g.byName[name]; ok {
		return &Symbol{g: g, id: id}
	}
	return g.add(name, kindTerm, named)
}

func (g *Grammar) add(name string, k kind, named bool) *Symbol {
	id := SymbolID(len(g.syms))
	hidden := isHidden(name)
	g.syms = append(g.syms, symRec{
		name:   name,
		kind:   k,
		named:  named && !hidden,
		hidden: hidden,
	})
	g.byName[name] = id
	return &Symbol{g: g, id: id}
}

func (g *Grammar) hidden(prefix string) *Rule {
	name := fmt.Sprintf("%s#%d", prefix, len(g.syms))
	s := g.add(name, kindNonterm, false)
	g.syms[s.id].hidden = true
	return &Rule{g: g, id: s.id}
}

func isHidden(name string) bool {
	return name == "" || name[0] == '_' || name[0] == '$'
}

func hasBytes(text string) Match {
	return func(src []byte, at int) int {
		if hasPrefix(src, at, text) {
			return len(text)
		}
		return 0
	}
}

func hasPrefix(src []byte, at int, text string) bool {
	if at+len(text) > len(src) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[at+i] != text[i] {
			return false
		}
	}
	return true
}
