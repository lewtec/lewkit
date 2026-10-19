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
	raw    *Form
	terms  map[string]Term
	rules  map[string]Rule
	builds map[string]func(SyntaxNode[Term, Rule]) (any, error)
	err    error
}

// Model is a compiled grammar that can read a token or a rule back off a node.
type Model[Term, Rule Enum] struct {
	language *Language
	terms    map[string]Term
	rules    map[string]Rule
	fields   map[string][]fieldBind
	builds   map[string]func(SyntaxNode[Term, Rule]) (any, error)
}

// Prepared is a declaration placed into a grammar, before its tables exist.
type Prepared[Term, Rule Enum] struct {
	grammar *Grammar
	terms   map[string]Term
	rules   map[string]Rule
	fields  map[string][]fieldBind
	builds  map[string]func(SyntaxNode[Term, Rule]) (any, error)
}

// SyntaxNode is a node in a parse that used a [Model]. Hidden rules invented
// while lowering are neither a token nor a rule.
type SyntaxNode[Term, Rule Enum] struct {
	Node
	model *Model[Term, Rule]
}

// Lit is an exact-text token. It has no enum value.
func (Syntax[Term, Rule]) Lit(text string) *Piece[Term, Rule] {
	return &Piece[Term, Rule]{raw: Lit(text)}
}

// Pat is a token of the caller's enum described by a pattern.
func (Syntax[Term, Rule]) Pat(kind Term, pattern Pattern) *Piece[Term, Rule] {
	raw := Tok(kind.String(), nil)
	raw.pattern = pattern
	raw.hasPattern = true
	if pattern.hasLex {
		raw.lex = pattern.lex
		raw.hasLex = true
	}
	piece := &Piece[Term, Rule]{raw: raw}
	piece.noteTerm(kind)
	return piece
}

// Term is a token of the caller's enum.
func (Syntax[Term, Rule]) Term(kind Term, match Match) *Piece[Term, Rule] {
	piece := &Piece[Term, Rule]{raw: Tok(kind.String(), match)}
	piece.noteTerm(kind)
	return piece
}

// Word is a token classified by a start rune and a continuation rune.
func (Syntax[Term, Rule]) Word(kind Term, start, continuation func(rune) bool) *Piece[Term, Rule] {
	piece := &Piece[Term, Rule]{raw: Word(kind.String(), start, continuation)}
	piece.noteTerm(kind)
	return piece
}

// Key is a keyword. word is the identifier token that shares its continuation.
func (Syntax[Term, Rule]) Key(kind, word Term) *Piece[Term, Rule] {
	piece := &Piece[Term, Rule]{raw: Key(kind.String(), word.String())}
	piece.noteTerm(kind)
	piece.noteTerm(word)
	return piece
}

// Keywords binds each term as a keyword of word. term.String is the spelling.
func (syntax Syntax[Term, Rule]) Keywords(word Term, terms ...Term) *Piece[Term, Rule] {
	parts := make([]*Piece[Term, Rule], 0, len(terms))
	for _, term := range terms {
		parts = append(parts, syntax.Key(term, word))
	}
	return syntax.Compose(parts...)
}

// Flag names a bool field set when piece is present.
func (syntax Syntax[Term, Rule]) Flag(piece *Piece[Term, Rule], name string) *Piece[Term, Rule] {
	return syntax.mark(piece, name, fieldFlag)
}

// Text names a string field filled from piece.
func (syntax Syntax[Term, Rule]) Text(piece *Piece[Term, Rule], name string) *Piece[Term, Rule] {
	return syntax.mark(piece, name, fieldText)
}

// Ref names a rule.
func (Syntax[Term, Rule]) Ref(name Rule) *Piece[Term, Rule] {
	piece := &Piece[Term, Rule]{raw: Ref(name.String())}
	piece.noteRule(name)
	return piece
}

// Eq binds a rule name to a body.
func (syntax Syntax[Term, Rule]) Eq(name Rule, body *Piece[Term, Rule]) *Piece[Term, Rule] {
	var rawBody *Form
	if body != nil {
		rawBody = body.raw
	}
	piece := &Piece[Term, Rule]{raw: Eq(name.String(), rawBody)}
	piece.noteRule(name)
	piece.join(body)
	return piece
}

// Seq is concatenation.
func (syntax Syntax[Term, Rule]) Seq(parts ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.gather(func(raw []*Form) *Form { return Seq(raw...) }, parts)
}

// Alt is a choice.
func (syntax Syntax[Term, Rule]) Alt(alternatives ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.gather(func(raw []*Form) *Form { return Alt(raw...) }, alternatives)
}

// Compose unions expressions.
func (syntax Syntax[Term, Rule]) Compose(parts ...*Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.gather(func(raw []*Form) *Form { return Compose(raw...) }, parts)
}

// Repeat is zero or more.
func (syntax Syntax[Term, Rule]) Repeat(piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Repeat(raw) })
}

// Repeat1 is one or more.
func (syntax Syntax[Term, Rule]) Repeat1(piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Repeat1(raw) })
}

// Opt is the expression or nothing.
func (syntax Syntax[Term, Rule]) Opt(piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Opt(raw) })
}

// Sep is zero or more items separated by separator.
func (syntax Syntax[Term, Rule]) Sep(item, separator *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.pair(item, separator, Sep)
}

// Sep1 is one or more items separated by separator.
func (syntax Syntax[Term, Rule]) Sep1(item, separator *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.pair(item, separator, Sep1)
}

// Left sets precedence and left associativity on an alternative.
func (syntax Syntax[Term, Rule]) Left(precedence int, piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Left(precedence, raw) })
}

// Right sets precedence and right associativity on an alternative.
func (syntax Syntax[Term, Rule]) Right(precedence int, piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Right(precedence, raw) })
}

// Prec sets precedence and leaves a tie as a fork.
func (syntax Syntax[Term, Rule]) Prec(precedence int, piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Prec(precedence, raw) })
}

// Dynamic sets the runtime precedence added when a fork covers a span.
func (syntax Syntax[Term, Rule]) Dynamic(precedence int, piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Dynamic(precedence, raw) })
}

// Lex sets lexical precedence on a token. A higher value wins a tie.
func (syntax Syntax[Term, Rule]) Lex(precedence int, piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Lex(precedence, raw) })
}

// Skip marks a token as trivia.
func (syntax Syntax[Term, Rule]) Skip(piece *Piece[Term, Rule]) *Piece[Term, Rule] {
	return syntax.wrap(piece, func(raw *Form) *Form { return Skip(raw) })
}

// Scan installs the whole-file scanner.
func (Syntax[Term, Rule]) Scan(scan ScanFunc) *Piece[Term, Rule] {
	return &Piece[Term, Rule]{raw: Scan(scan)}
}

// Build attaches the function that turns this rule into a value.
// The piece has to be an [Eq]. Constructors still return a new piece.
func (piece *Piece[Term, Rule]) Build(fn func(SyntaxNode[Term, Rule]) (any, error)) *Piece[Term, Rule] {
	if piece == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: build belongs on a rule")}
	}
	if piece.err != nil {
		return piece
	}
	if fn == nil || piece.raw == nil || piece.raw.kind != formEq || piece.raw.name == "" {
		out := *piece
		out.err = errs.Wrap(errs.ErrInvalid, "build: build belongs on a rule")
		return &out
	}
	out := *piece
	out.builds = make(map[string]func(SyntaxNode[Term, Rule]) (any, error), len(piece.builds)+1)
	for name, build := range piece.builds {
		out.builds[name] = build
	}
	if _, ok := out.builds[piece.raw.name]; ok {
		out.err = errs.Wrap(errs.ErrInvalid, "build: rule "+piece.raw.name+" has two builds")
		return &out
	}
	out.builds[piece.raw.name] = fn
	return &out
}

// Start selects the root rule.
func (syntax Syntax[Term, Rule]) Start(name Rule) *Piece[Term, Rule] {
	piece := &Piece[Term, Rule]{raw: Start(name.String())}
	piece.err = checkEnum(name)
	return piece
}

// Compile places the piece and builds its parse tables.
func (syntax Syntax[Term, Rule]) Compile(name string, piece *Piece[Term, Rule]) (*Model[Term, Rule], error) {
	prepared, err := syntax.Prepare(name, piece)
	if err != nil {
		return nil, err
	}
	return prepared.Compile()
}

// Prepare places the piece into a grammar and keeps the record fields.
// [Prepared.Compile] builds the tables. [Prepared.Load] reads tables that
// were generated from the same declaration.
func (Syntax[Term, Rule]) Prepare(name string, piece *Piece[Term, Rule]) (*Prepared[Term, Rule], error) {
	if piece == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty grammar")
	}
	if piece.err != nil {
		return nil, piece.err
	}
	grammar, err := Build(name, piece.raw)
	if err != nil {
		return nil, err
	}
	fields, err := expressionFields(piece.raw)
	if err != nil {
		return nil, err
	}
	return &Prepared[Term, Rule]{
		grammar: grammar,
		terms:   piece.terms,
		rules:   piece.rules,
		fields:  fields,
		builds:  piece.builds,
	}, nil
}

// Compile builds the parse tables for the declaration.
func (prepared *Prepared[Term, Rule]) Compile() (*Model[Term, Rule], error) {
	if prepared == nil || prepared.grammar == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty grammar")
	}
	language, err := prepared.grammar.Compile()
	if err != nil {
		return nil, err
	}
	return prepared.model(language), nil
}

// TableBytes compiles a declaration into the LR(1) table.
// err is the error from placing that declaration.
func TableBytes[Term, Rule Enum](prepared *Prepared[Term, Rule], err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	model, err := prepared.Compile()
	if err != nil {
		return nil, err
	}
	return model.Table()
}

// Scanner installs the runtime scanner when the declaration did not name one.
// The table does not store it. A declaration that already has a scanner
// keeps that one.
func (prepared *Prepared[Term, Rule]) Scanner(fn ScanFunc) error {
	if prepared == nil || prepared.grammar == nil {
		return errs.Wrap(errs.ErrInvalid, "build: empty grammar")
	}
	if prepared.grammar.scan != nil {
		return errs.Wrap(errs.ErrInvalid, "build: two scanners")
	}
	if fn == nil {
		return errs.Wrap(errs.ErrInvalid, "build: empty scanner")
	}
	prepared.grammar.Scanner(fn)
	return nil
}

// Load reads a table from [Model.Table]. The declaration supplies matchers.
func (prepared *Prepared[Term, Rule]) Load(table []byte) (*Model[Term, Rule], error) {
	if prepared == nil || prepared.grammar == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "build: empty grammar")
	}
	language, err := prepared.grammar.Load(table)
	if err != nil {
		return nil, err
	}
	return prepared.model(language), nil
}

func (prepared *Prepared[Term, Rule]) model(language *Language) *Model[Term, Rule] {
	return &Model[Term, Rule]{
		language: language,
		terms:    prepared.terms,
		rules:    prepared.rules,
		fields:   prepared.fields,
		builds:   prepared.builds,
	}
}

// Table encodes the automaton. Matchers stay in the declaration.
func (model *Model[Term, Rule]) Table() ([]byte, error) {
	if model == nil || model.language == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: nil language")
	}
	return model.language.MarshalTable()
}

// Parse reads src. The returned node is the root, or the zero node on error.
func (model *Model[Term, Rule]) Parse(src []byte) (SyntaxNode[Term, Rule], error) {
	if model == nil || model.language == nil {
		return SyntaxNode[Term, Rule]{}, errs.Wrap(errs.ErrInvalid, "parse: nil language")
	}
	tree, err := model.language.Parse(src)
	if err != nil {
		return SyntaxNode[Term, Rule]{}, err
	}
	return SyntaxNode[Term, Rule]{Node: tree.Root(), model: model}, nil
}

// Term reports the token enum value of a terminal.
func (node SyntaxNode[Term, Rule]) Term() (Term, bool) {
	var zero Term
	if !node.terminal() {
		return zero, false
	}
	value, ok := node.model.terms[node.Name()]
	return value, ok
}

// Rule reports the rule enum value of a visible rule.
func (node SyntaxNode[Term, Rule]) Rule() (Rule, bool) {
	var zero Rule
	if !node.Valid() || node.model == nil || node.terminal() {
		return zero, false
	}
	value, ok := node.model.rules[node.Name()]
	return value, ok
}

// Child is the ith child in this same typed tree.
func (node SyntaxNode[Term, Rule]) Child(index int) SyntaxNode[Term, Rule] {
	return SyntaxNode[Term, Rule]{Node: node.Node.Child(index), model: node.model}
}

func (node SyntaxNode[Term, Rule]) terminal() bool {
	if !node.Valid() || node.model == nil {
		return false
	}
	symbol := node.tree.nodes[node.id].kind
	return node.tree.lang.syms[symbol].terminal
}

func (syntax Syntax[Term, Rule]) wrap(piece *Piece[Term, Rule], apply func(*Form) *Form) *Piece[Term, Rule] {
	if piece == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: empty expression")}
	}
	return &Piece[Term, Rule]{raw: apply(piece.raw), terms: piece.terms, rules: piece.rules, builds: piece.builds, err: piece.err}
}

func (syntax Syntax[Term, Rule]) mark(piece *Piece[Term, Rule], name string, kind uint8) *Piece[Term, Rule] {
	if piece == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: empty expression")}
	}
	if name == "" {
		out := *piece
		out.err = errs.Wrap(errs.ErrInvalid, "build: field needs a name and a symbol")
		return &out
	}
	raw := &Form{kind: formMark, field: name, fieldAs: kind, kids: []*Form{piece.raw}}
	return &Piece[Term, Rule]{raw: raw, terms: piece.terms, rules: piece.rules, builds: piece.builds, err: piece.err}
}

func (syntax Syntax[Term, Rule]) pair(item, separator *Piece[Term, Rule], apply func(*Form, *Form) *Form) *Piece[Term, Rule] {
	if item == nil || separator == nil {
		return &Piece[Term, Rule]{err: errs.Wrap(errs.ErrInvalid, "build: separator needs an item and a separator")}
	}
	out := &Piece[Term, Rule]{raw: apply(item.raw, separator.raw)}
	out.join(item)
	out.join(separator)
	return out
}

func (syntax Syntax[Term, Rule]) gather(apply func([]*Form) *Form, parts []*Piece[Term, Rule]) *Piece[Term, Rule] {
	raw := make([]*Form, 0, len(parts))
	out := &Piece[Term, Rule]{}
	for _, piece := range parts {
		if piece == nil {
			continue
		}
		raw = append(raw, piece.raw)
		out.join(piece)
	}
	out.raw = apply(raw)
	return out
}

func (piece *Piece[Term, Rule]) noteTerm(kind Term) {
	if piece.err == nil {
		piece.err = checkEnum(kind)
	}
	piece.terms = putEnum(piece.terms, kind.String(), kind, &piece.err)
}

func (piece *Piece[Term, Rule]) noteRule(name Rule) {
	if piece.err == nil {
		piece.err = checkEnum(name)
	}
	piece.rules = putEnum(piece.rules, name.String(), name, &piece.err)
}

func (piece *Piece[Term, Rule]) join(other *Piece[Term, Rule]) {
	if other == nil {
		return
	}
	if piece.err == nil {
		piece.err = other.err
	}
	for name, value := range other.terms {
		piece.terms = putEnum(piece.terms, name, value, &piece.err)
	}
	for name, value := range other.rules {
		piece.rules = putEnum(piece.rules, name, value, &piece.err)
	}
	for name, build := range other.builds {
		if piece.builds != nil {
			if _, ok := piece.builds[name]; ok {
				if piece.err == nil {
					piece.err = errs.Wrap(errs.ErrInvalid, "build: rule "+name+" has two builds")
				}
				continue
			}
		} else {
			piece.builds = map[string]func(SyntaxNode[Term, Rule]) (any, error){}
		}
		piece.builds[name] = build
	}
}

func putEnum[T Enum](values map[string]T, key string, value T, err *error) map[string]T {
	if values == nil {
		values = map[string]T{}
	}
	if old, ok := values[key]; ok && old != value {
		if err != nil && *err == nil {
			*err = errs.Wrap(errs.ErrInvalid, "build: symbol "+key+" has two values")
		}
		return values
	}
	values[key] = value
	return values
}

type validEnum interface {
	IsValid() bool
}

func checkEnum[T Enum](value T) error {
	if value.String() == "" {
		return errs.Wrap(errs.ErrInvalid, "build: empty symbol")
	}
	if checker, ok := any(value).(validEnum); ok && !checker.IsValid() {
		return errs.Wrap(errs.ErrInvalid, "build: invalid symbol "+value.String())
	}
	return nil
}
