package parse

import "github.com/lewtec/lewkit/x/errs"

// Form is one grammar expression. Constructors return a new form and do not
// mutate an existing one. [Compose] unions expressions: two [Eq] bindings of
// one name become alternatives of that rule. [Build] places the expression
// into a [Grammar]. A form does not remember which grammar it was built into.
type Form struct {
	kind       formKind
	name       string
	text       string
	word       string
	kids       []*Form
	match      Match
	start      func(rune) bool
	cont       func(rune) bool
	scan       ScanFunc
	prec       int
	hasPrec    bool
	assoc      Assoc
	dyn        int
	hasDyn     bool
	lex        int
	hasLex     bool
	pattern    Pattern
	hasPattern bool
	field      string
	fieldAs    uint8
}

type formKind uint8

const (
	formRef formKind = iota + 1
	formLit
	formTok
	formWord
	formKey
	formSeq
	formAlt
	formRepeat
	formRepeat1
	formOpt
	formSep
	formSep1
	formPrec
	formEq
	formSkip
	formScan
	formStart
	formCompose
	formMark
)

// Ref names a rule or a token. The name is resolved when the form is built.
func Ref(name string) *Form {
	return &Form{kind: formRef, name: name}
}

// Lit is an exact-text token. The same text is one symbol in a grammar.
func Lit(text string) *Form {
	return &Form{kind: formLit, text: text}
}

// Tok is a named token with a matcher. A nil matcher is filled by a scanner.
func Tok(name string, match Match) *Form {
	return &Form{kind: formTok, name: name, match: match}
}

// Word is a named token classified by a start rune and a continuation rune.
func Word(name string, start, cont func(rune) bool) *Form {
	return &Form{kind: formWord, name: name, start: start, cont: cont}
}

// Key is a keyword. word is the name of the identifier token that shares
// its continuation test, so the keyword does not match inside a longer word.
func Key(text, word string) *Form {
	return &Form{kind: formKey, text: text, word: word}
}

// Seq is concatenation.
func Seq(parts ...*Form) *Form {
	return &Form{kind: formSeq, kids: keep(parts)}
}

// Alt is a choice. Each child is one alternative.
func Alt(alts ...*Form) *Form {
	return &Form{kind: formAlt, kids: keep(alts)}
}

// Repeat is zero or more. The rule is hidden and flattened into its parent.
func Repeat(f *Form) *Form {
	return &Form{kind: formRepeat, kids: []*Form{f}}
}

// Repeat1 is one or more.
func Repeat1(f *Form) *Form {
	return &Form{kind: formRepeat1, kids: []*Form{f}}
}

// Opt is the expression or nothing.
func Opt(f *Form) *Form {
	return &Form{kind: formOpt, kids: []*Form{f}}
}

// Sep is zero or more items separated by sep.
func Sep(item, sep *Form) *Form {
	return &Form{kind: formSep, kids: []*Form{item, sep}}
}

// Sep1 is one or more items separated by sep.
func Sep1(item, sep *Form) *Form {
	return &Form{kind: formSep1, kids: []*Form{item, sep}}
}

// Left sets precedence and left associativity on an alternative.
func Left(n int, f *Form) *Form {
	return precForm(n, AssocLeft, f)
}

// Right sets precedence and right associativity on an alternative.
func Right(n int, f *Form) *Form {
	return precForm(n, AssocRight, f)
}

// Prec sets precedence and leaves a tie as a fork.
func Prec(n int, f *Form) *Form {
	return precForm(n, AssocNone, f)
}

// Dynamic sets the runtime precedence added when a fork covers a span.
func Dynamic(n int, f *Form) *Form {
	return &Form{kind: formPrec, dyn: n, hasDyn: true, kids: []*Form{f}}
}

// Lex sets the lexical precedence of a token. A higher value wins when two
// tokens match the same bytes. Keywords use 2. Identifiers use 0.
func Lex(n int, f *Form) *Form {
	if f == nil {
		return nil
	}
	g := *f
	g.lex = n
	g.hasLex = true
	return &g
}

// Eq binds a name to a body. The body is one alternative, or an [Alt].
func Eq(name string, body *Form) *Form {
	return &Form{kind: formEq, name: name, kids: []*Form{body}}
}

// Skip marks a token as trivia. The lexer consumes it and leaves it out of the tree.
func Skip(f *Form) *Form {
	return &Form{kind: formSkip, kids: []*Form{f}}
}

// Scan installs the whole-file scanner. A second scanner in one [Compose] is an error.
func Scan(fn ScanFunc) *Form {
	if fn == nil {
		return nil
	}
	return &Form{kind: formScan, scan: fn}
}

// Start selects the root rule.
func Start(name string) *Form {
	return &Form{kind: formStart, name: name}
}

// Compose unions expressions. Rule equations with one name append alternatives
// in left-to-right order. Token matchers do not replace a matcher already set.
func Compose(parts ...*Form) *Form {
	return &Form{kind: formCompose, kids: keep(parts)}
}

// Build places f into a new grammar called name.
func Build(name string, f *Form) (*Grammar, error) {
	g := New(name)
	if f == nil {
		return g, nil
	}
	var c coll
	if err := walk(f, &c, map[*Form]bool{}); err != nil {
		return nil, err
	}
	p := &placer{g: g, seen: map[*Form]*Symbol{}}
	if err := p.tokens(c.toks); err != nil {
		return nil, err
	}
	for _, s := range c.skips {
		if err := p.skip(s); err != nil {
			return nil, err
		}
	}
	if c.scanSet {
		g.scan = c.scan
	}
	for _, r := range c.rules {
		if err := p.placeAlts(g.Rule(r.name), r.body, precInfo{}); err != nil {
			return nil, err
		}
	}
	if c.startSet {
		g.Start(c.start)
	}
	return g, nil
}

func precForm(n int, assoc Assoc, f *Form) *Form {
	return &Form{kind: formPrec, prec: n, hasPrec: true, assoc: assoc, kids: []*Form{f}}
}

func keep(in []*Form) []*Form {
	out := make([]*Form, 0, len(in))
	for _, f := range in {
		if f != nil {
			out = append(out, f)
		}
	}
	return out
}

type binding struct {
	name string
	body *Form
}

type fieldBind struct {
	rule   string
	symbol string
	name   string
	kind   uint8
}

const (
	fieldFlag uint8 = 1
	fieldText uint8 = 2
)

type coll struct {
	toks     []*Form
	skips    []*Form
	rules    []binding
	fields   []fieldBind
	rule     string
	scan     ScanFunc
	scanSet  bool
	start    string
	startSet bool
}

func walk(f *Form, c *coll, seen map[*Form]bool) error {
	top := f != nil && f.kind == formSkip
	return walkAt(f, c, seen, top)
}

func walkAt(f *Form, c *coll, seen map[*Form]bool, top bool) error {
	if f == nil || seen[f] {
		return nil
	}
	seen[f] = true
	switch f.kind {
	case formEq:
		if f.name == "" || len(f.kids) != 1 || f.kids[0] == nil {
			return errs.Wrap(errs.ErrInvalid, "build: rule needs a name and a body")
		}
		c.rules = append(c.rules, binding{name: f.name, body: f.kids[0]})
		prev := c.rule
		c.rule = f.name
		err := walkAt(f.kids[0], c, seen, false)
		c.rule = prev
		return err
	case formSkip:
		if len(f.kids) != 1 || f.kids[0] == nil {
			return errs.Wrap(errs.ErrInvalid, "build: skip needs a token")
		}
		if top {
			c.skips = append(c.skips, f.kids[0])
		}
		return walkAt(f.kids[0], c, seen, false)
	case formCompose:
		for _, k := range f.kids {
			childTop := k != nil && k.kind == formSkip
			if err := walkAt(k, c, seen, childTop); err != nil {
				return err
			}
		}
		return nil
	case formMark:
		if f.field == "" || len(f.kids) != 1 || f.kids[0] == nil {
			return errs.Wrap(errs.ErrInvalid, "build: field needs a name and a symbol")
		}
		if c.rule == "" {
			return errs.Wrap(errs.ErrInvalid, "build: field belongs on a rule")
		}
		symbol := symbolName(f.kids[0])
		if symbol == "" {
			return errs.Wrap(errs.ErrInvalid, "build: field needs a symbol")
		}
		c.fields = append(c.fields, fieldBind{rule: c.rule, symbol: symbol, name: f.field, kind: f.fieldAs})
	case formScan:
		if c.scanSet {
			return errs.Wrap(errs.ErrInvalid, "build: two scanners")
		}
		c.scan, c.scanSet = f.scan, true
	case formStart:
		if f.name == "" {
			return errs.Wrap(errs.ErrInvalid, "build: empty start")
		}
		if c.startSet && c.start != f.name {
			return errs.Wrap(errs.ErrInvalid, "build: two start rules")
		}
		c.start, c.startSet = f.name, true
	case formTok, formWord:
		if f.name == "" {
			return errs.Wrap(errs.ErrInvalid, "build: empty token name")
		}
		c.toks = append(c.toks, f)
	case formKey:
		if f.text == "" || f.word == "" {
			return errs.Wrap(errs.ErrInvalid, "build: keyword needs text and a word")
		}
		c.toks = append(c.toks, f)
	case formRef:
		if f.name == "" {
			return errs.Wrap(errs.ErrInvalid, "build: empty name")
		}
	case formRepeat, formRepeat1, formOpt, formPrec:
		if len(f.kids) != 1 || f.kids[0] == nil {
			return errs.Wrap(errs.ErrInvalid, "build: expression needs one child")
		}
	case formSep, formSep1:
		if len(f.kids) != 2 || f.kids[0] == nil || f.kids[1] == nil {
			return errs.Wrap(errs.ErrInvalid, "build: separator needs an item and a separator")
		}
	case formLit, formSeq, formAlt:
	default:
		return errs.Wrap(errs.ErrInvalid, "build: unknown expression")
	}
	for _, k := range f.kids {
		if err := walkAt(k, c, seen, false); err != nil {
			return err
		}
	}
	return nil
}

func expressionFields(f *Form) (map[string][]fieldBind, error) {
	var c coll
	if err := walk(f, &c, map[*Form]bool{}); err != nil {
		return nil, err
	}
	out := map[string][]fieldBind{}
	for _, field := range c.fields {
		out[field.rule] = append(out[field.rule], field)
	}
	return out, nil
}

func symbolName(f *Form) string {
	for f != nil {
		f, _ = peel(f)
		if f == nil {
			return ""
		}
		switch f.kind {
		case formMark, formSkip, formOpt, formRepeat, formRepeat1:
			if len(f.kids) == 0 {
				return ""
			}
			f = f.kids[0]
		case formTok, formWord, formRef:
			return f.name
		case formKey, formLit:
			return f.text
		default:
			return ""
		}
	}
	return ""
}

type precInfo struct {
	prec   int
	has    bool
	assoc  Assoc
	dyn    int
	hasDyn bool
}

func peel(f *Form) (*Form, precInfo) {
	var info precInfo
	for f != nil && f.kind == formPrec {
		if !info.has && f.hasPrec {
			info.prec, info.has, info.assoc = f.prec, true, f.assoc
		}
		if !info.hasDyn && f.hasDyn {
			info.dyn, info.hasDyn = f.dyn, true
		}
		if len(f.kids) == 0 {
			return nil, info
		}
		f = f.kids[0]
	}
	return f, info
}

func (info precInfo) with(got precInfo) precInfo {
	if got.has {
		info.prec, info.has, info.assoc = got.prec, true, got.assoc
	}
	if got.hasDyn {
		info.dyn, info.hasDyn = got.dyn, true
	}
	return info
}

func (info precInfo) apply(p *Prod) {
	if p == nil {
		return
	}
	if info.has {
		switch info.assoc {
		case AssocLeft:
			p.Left(info.prec)
		case AssocRight:
			p.Right(info.prec)
		default:
			p.Prec(info.prec)
		}
	}
	if info.hasDyn {
		p.Dynamic(info.dyn)
	}
}

type placer struct {
	g    *Grammar
	seen map[*Form]*Symbol
}

func (p *placer) tokens(toks []*Form) error {
	for _, t := range toks {
		if t.kind == formKey {
			continue
		}
		if _, err := p.placeToken(t); err != nil {
			return err
		}
	}
	for _, t := range toks {
		if t.kind != formKey {
			continue
		}
		if _, err := p.placeToken(t); err != nil {
			return err
		}
	}
	return nil
}

func (p *placer) placeToken(f *Form) (*Symbol, error) {
	if f == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty token")
	}
	if s, ok := p.seen[f]; ok {
		return s, nil
	}
	switch f.kind {
	case formTok:
		s := p.g.Token(f.name)
		rec := &p.g.syms[s.id]
		if f.hasPattern || f.match != nil {
			if rec.match != nil || rec.matchPrev != nil {
				return nil, errs.Wrap(errs.ErrInvalid, "build: token "+f.name+" has two matchers")
			}
		}
		if f.hasPattern {
			built := f.pattern.build(p.g.nameOf)
			if built.prev != nil {
				rec.matchPrev = built.prev
			} else {
				rec.match = built.match
			}
		} else if f.match != nil {
			rec.match = f.match
		}
		if f.hasLex {
			rec.lexPrec = f.lex
		}
		p.seen[f] = s
		return s, nil
	case formWord:
		s := p.g.Token(f.name)
		rec := &p.g.syms[s.id]
		if rec.match != nil || rec.matchPrev != nil || rec.cont != nil {
			return nil, errs.Wrap(errs.ErrInvalid, "build: token "+f.name+" has two matchers")
		}
		s.Word(f.start, f.cont)
		if f.hasLex {
			p.g.syms[s.id].lexPrec = f.lex
		}
		p.seen[f] = s
		return s, nil
	case formKey:
		s := p.g.Keyword(f.text, p.g.Sym(f.word))
		if f.hasLex {
			p.g.syms[s.id].lexPrec = f.lex
		}
		p.seen[f] = s
		return s, nil
	default:
		return nil, errs.Wrap(errs.ErrInvalid, "build: not a token")
	}
}

func (p *placer) skip(f *Form) error {
	s, err := p.placeSymbol(f)
	if err != nil {
		return err
	}
	if p.g.syms[s.id].extra {
		return nil
	}
	p.g.Skip(s)
	return nil
}

func (p *placer) placeAlts(rule *Rule, f *Form, inh precInfo) error {
	body, got := peel(f)
	info := inh.with(got)
	if body == nil {
		return errs.Wrap(errs.ErrInvalid, "build: empty expression")
	}
	switch body.kind {
	case formAlt:
		if len(body.kids) == 0 {
			return errs.Wrap(errs.ErrInvalid, "build: empty choice")
		}
		for _, k := range body.kids {
			if err := p.placeAlts(rule, k, info); err != nil {
				return err
			}
		}
		return nil
	case formSeq:
		syms := make([]*Symbol, 0, len(body.kids))
		for _, k := range body.kids {
			s, err := p.placeSymbol(k)
			if err != nil {
				return err
			}
			syms = append(syms, s)
		}
		info.apply(rule.Seq(syms...))
		return nil
	case formCompose, formSkip, formScan, formStart:
		return errs.Wrap(errs.ErrInvalid, "build: expression is not an alternative")
	default:
		s, err := p.placeSymbol(body)
		if err != nil {
			return err
		}
		info.apply(rule.Seq(s))
		return nil
	}
}

func (p *placer) placeSymbol(f *Form) (*Symbol, error) {
	if f == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty expression")
	}
	switch f.kind {
	case formRef:
		if f.name == "" {
			return nil, errs.Wrap(errs.ErrInvalid, "build: empty name")
		}
		return p.g.Sym(f.name), nil
	case formLit:
		return p.g.Literal(f.text), nil
	case formTok, formWord, formKey:
		return p.placeToken(f)
	case formRepeat, formRepeat1, formOpt:
		if len(f.kids) != 1 {
			return nil, errs.Wrap(errs.ErrInvalid, "build: expression needs one child")
		}
		s, err := p.placeSymbol(f.kids[0])
		if err != nil {
			return nil, err
		}
		switch f.kind {
		case formRepeat:
			return p.g.Repeat(s), nil
		case formRepeat1:
			return p.g.Repeat1(s), nil
		default:
			return p.g.Opt(s), nil
		}
	case formSep, formSep1:
		if len(f.kids) != 2 {
			return nil, errs.Wrap(errs.ErrInvalid, "build: separator needs an item and a separator")
		}
		item, err := p.placeSymbol(f.kids[0])
		if err != nil {
			return nil, err
		}
		sep, err := p.placeSymbol(f.kids[1])
		if err != nil {
			return nil, err
		}
		if f.kind == formSep {
			return p.g.Sep(item, sep), nil
		}
		return p.g.Sep1(item, sep), nil
	case formSeq:
		r := p.g.hidden("$seq")
		if err := p.placeAlts(r, f, precInfo{}); err != nil {
			return nil, err
		}
		return &Symbol{g: p.g, id: r.id}, nil
	case formAlt:
		r := p.g.hidden("$alt")
		if err := p.placeAlts(r, f, precInfo{}); err != nil {
			return nil, err
		}
		return &Symbol{g: p.g, id: r.id}, nil
	case formEq:
		if f.name == "" {
			return nil, errs.Wrap(errs.ErrInvalid, "build: empty name")
		}
		return p.g.Sym(f.name), nil
	case formPrec:
		return nil, errs.Wrap(errs.ErrInvalid, "build: precedence belongs on an alternative")
	case formSkip, formMark:
		if len(f.kids) != 1 || f.kids[0] == nil {
			return nil, errs.Wrap(errs.ErrInvalid, "build: empty expression")
		}
		return p.placeSymbol(f.kids[0])
	case formScan, formStart, formCompose:
		return nil, errs.Wrap(errs.ErrInvalid, "build: expression is not a symbol")
	default:
		return nil, errs.Wrap(errs.ErrInvalid, "build: unknown expression")
	}
}
