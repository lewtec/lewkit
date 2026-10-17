package parse

import "github.com/lewtec/lewkit/x/errs"

// Enum is a closed token or rule set. String is the symbol name stored
// on the grammar. A [Syntax] carries one token enum and one rule enum
// and stores nothing itself.
type Enum interface {
	~int | ~string
	String() string
}

// Syntax carries the token and rule enums for a grammar. Its methods
// return new pieces. The value is empty and is not a place to store rules.
type Syntax[Term, Rule Enum] struct{}

// Piece is one typed grammar expression.
type Piece[Term, Rule Enum] struct {
	raw   *Form
	terms map[string]Term
	rules map[string]Rule
	err   error
}

// Model is a compiled grammar that can read a token or a rule back off a node.
type Model[Term, Rule Enum] struct {
	lang  *Language
	terms map[string]Term
	rules map[string]Rule
}

// Syn is a node in a parse that used a [Model]. Hidden rules invented
// while lowering are neither a token nor a rule.
type Syn[Term, Rule Enum] struct {
	Node
	model *Model[Term, Rule]
}

// Lit is an exact-text token. It has no enum value.
func (Syntax[Term, Rule]) Lit(text string) *Piece[Term, Rule] {
	return &Piece[Term, Rule]{raw: Lit(text)}
}

// Term is a token of the caller's enum.
func (Syntax[Term, Rule]) Term(kind Term, match Match) *Piece[Term, Rule] {
	p := &Piece[Term, Rule]{raw: Tok(kind.String(), match)}
	p.noteTerm(kind)
	return p
}

// Word is a token classified by a start rune and a continuation rune.
func (Syntax[Term, Rule]) Word(kind Term, start, cont func(rune) bool) *Piece[Term, Rule] {
	p := &Piece[Term, Rule]{raw: Word(kind.String(), start, cont)}
	p.noteTerm(kind)
	return p
}

// Key is a keyword. word is the identifier token that shares its continuation.
func (Syntax[Term, Rule]) Key(kind, word Term) *Piece[Term, Rule] {
	p := &Piece[Term, Rule]{raw: Key(kind.String(), word.String())}
	p.noteTerm(kind)
	p.noteTerm(word)
	return p
}

// Ref names a rule.
func (Syntax[Term, Rule]) Ref(name Rule) *Piece[Term, Rule] {
	p := &Piece[Term, Rule]{raw: Ref(name.String())}
	p.noteRule(name)
	return p
}

// Eq binds a rule name to a body.
func (s Syntax[Term, Rule]) Eq(name Rule, body *Piece[Term, Rule]) *Piece[Term, Rule] {
	var rawBody *Form
	if body != nil {
		rawBody = body.raw
	}
	p := &Piece[Term, Rule]{raw: Eq(name.String(), rawBody)}
	p.noteRule(name)
	p.join(body)
	return p
}

// Seq is concatenation.
func (s Syntax[Term, Rule]) Seq(parts ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.gather(func(raw []*Form) *Form { return Seq(raw...) }, parts)
}

// Alt is a choice.
func (s Syntax[Term, Rule]) Alt(alts ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.gather(func(raw []*Form) *Form { return Alt(raw...) }, alts)
}

// Compose unions expressions.
func (s Syntax[Term, Rule]) Compose(parts ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.gather(func(raw []*Form) *Form { return Compose(raw...) }, parts)
}

// Repeat is zero or more.
func (s Syntax[Term, Rule]) Repeat(p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Repeat(raw) })
}

// Repeat1 is one or more.
func (s Syntax[Term, Rule]) Repeat1(p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Repeat1(raw) })
}

// Opt is the expression or nothing.
func (s Syntax[Term, Rule]) Opt(p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Opt(raw) })
}

// Sep is zero or more items separated by sep.
func (s Syntax[Term, Rule]) Sep(item, sep *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.pair(item, sep, Sep)
}

// Sep1 is one or more items separated by sep.
func (s Syntax[Term, Rule]) Sep1(item, sep *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.pair(item, sep, Sep1)
}

// Left sets precedence and left associativity on an alternative.
func (s Syntax[Term, Rule]) Left(n int, p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Left(n, raw) })
}

// Right sets precedence and right associativity on an alternative.
func (s Syntax[Term, Rule]) Right(n int, p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Right(n, raw) })
}

// Prec sets precedence and leaves a tie as a fork.
func (s Syntax[Term, Rule]) Prec(n int, p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Prec(n, raw) })
}

// Dynamic sets the runtime precedence added when a fork covers a span.
func (s Syntax[Term, Rule]) Dynamic(n int, p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Dynamic(n, raw) })
}

// Lex sets lexical precedence on a token. A higher value wins a tie.
func (s Syntax[Term, Rule]) Lex(n int, p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Lex(n, raw) })
}

// Skip marks a token as trivia.
func (s Syntax[Term, Rule]) Skip(p *Piece[Term, Rule]) *Piece[Term, Rule] {
	return s.wrap(p, func(raw *Form) *Form { return Skip(raw) })
}

// Scan installs the whole-file scanner.
func (Syntax[Term, Rule]) Scan(fn ScanFunc) *Piece[Term, Rule] {
	return &Piece[Term, Rule]{raw: Scan(fn)}
}

// Start selects the root rule.
func (s Syntax[Term, Rule]) Start(name Rule) *Piece[Term, Rule] {
	p := &Piece[Term, Rule]{raw: Start(name.String())}
	p.err = enumOK(name)
	return p
}

// Compile places the piece and builds its parse tables.
func (Syntax[Term, Rule]) Compile(name string, p *Piece[Term, Rule]) (*Model[Term, Rule], error) {
	if p == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty grammar")
	}
	if p.err != nil {
		return nil, p.err
	}
	g, err := Build(name, p.raw)
	if err != nil {
		return nil, err
	}
	lang, err := g.Compile()
	if err != nil {
		return nil, err
	}
	return &Model[Term, Rule]{lang: lang, terms: p.terms, rules: p.rules}, nil
}

// Parse reads src. The returned node is the root, or the zero node on error.
func (m *Model[Term, Rule]) Parse(src []byte) (Syn[Term, Rule], error) {
	if m == nil || m.lang == nil {
		return Syn[Term, Rule]{}, errs.Wrap(errs.ErrInvalid, "parse: nil language")
	}
	tree, err := m.lang.Parse(src)
	if err != nil {
		return Syn[Term, Rule]{}, err
	}
	return Syn[Term, Rule]{Node: tree.Root(), model: m}, nil
}

// Term reports the token enum value of a terminal.
func (n Syn[Term, Rule]) Term() (Term, bool) {
	var zero Term
	if !n.terminal() {
		return zero, false
	}
	v, ok := n.model.terms[n.Name()]
	return v, ok
}

// Rule reports the rule enum value of a visible rule.
func (n Syn[Term, Rule]) Rule() (Rule, bool) {
	var zero Rule
	if !n.Valid() || n.model == nil || n.terminal() {
		return zero, false
	}
	v, ok := n.model.rules[n.Name()]
	return v, ok
}

// Child is the ith child in this same typed tree.
func (n Syn[Term, Rule]) Child(i int) Syn[Term, Rule] {
	return Syn[Term, Rule]{Node: n.Node.Child(i), model: n.model}
}

func (n Syn[Term, Rule]) terminal() bool {
	if !n.Valid() || n.model == nil {
		return false
	}
	id := n.tree.nodes[n.id].kind
	return n.tree.lang.syms[id].terminal
}

func (s Syntax[Term, Rule]) wrap(p *Piece[Term, Rule], fn func(*Form) *Form) *Piece[Term, Rule] {
	if p == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: empty expression")}
	}
	out := &Piece[Term, Rule]{raw: fn(p.raw), terms: p.terms, rules: p.rules, err: p.err}
	return out
}

func (s Syntax[Term, Rule]) pair(a, b *Piece[Term, Rule], fn func(*Form, *Form) *Form) *Piece[Term, Rule] {
	if a == nil || b == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: separator needs an item and a separator")}
	}
	out := &Piece[Term, Rule]{raw: fn(a.raw, b.raw)}
	out.join(a)
	out.join(b)
	return out
}

func (s Syntax[Term, Rule]) gather(fn func([]*Form) *Form, parts []*Piece[Term, Rule]) *Piece[Term, Rule] {
	raw := make([]*Form, 0, len(parts))
	out := &Piece[Term, Rule]{}
	for _, p := range parts {
		if p == nil {
			continue
		}
		raw = append(raw, p.raw)
		out.join(p)
	}
	out.raw = fn(raw)
	return out
}

func (p *Piece[Term, Rule]) noteTerm(kind Term) {
	if p.err == nil {
		p.err = enumOK(kind)
	}
	p.terms = putEnum(p.terms, kind.String(), kind, &p.err)
}

func (p *Piece[Term, Rule]) noteRule(name Rule) {
	if p.err == nil {
		p.err = enumOK(name)
	}
	p.rules = putEnum(p.rules, name.String(), name, &p.err)
}

func (p *Piece[Term, Rule]) join(q *Piece[Term, Rule]) {
	if q == nil {
		return
	}
	if p.err == nil {
		p.err = q.err
	}
	for k, v := range q.terms {
		p.terms = putEnum(p.terms, k, v, &p.err)
	}
	for k, v := range q.rules {
		p.rules = putEnum(p.rules, k, v, &p.err)
	}
}

func putEnum[T Enum](m map[string]T, key string, v T, err *error) map[string]T {
	if m == nil {
		m = map[string]T{}
	}
	if old, ok := m[key]; ok && old != v {
		if err != nil && *err == nil {
			*err = errs.Wrap(errs.ErrInvalid, "build: symbol "+key+" has two values")
		}
		return m
	}
	m[key] = v
	return m
}

type validEnum interface {
	IsValid() bool
}

func enumOK[T Enum](v T) error {
	if v.String() == "" {
		return errs.Wrap(errs.ErrInvalid, "build: empty symbol")
	}
	if c, ok := any(v).(validEnum); ok && !c.IsValid() {
		return errs.Wrap(errs.ErrInvalid, "build: invalid symbol "+v.String())
	}
	return nil
}
