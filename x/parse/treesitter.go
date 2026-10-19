package parse

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
)

// TreeSitterInfo is what FromTreeSitter kept as a note while loading.
// Externals are scanner tokens with no matcher. Notes are the other gaps.
type TreeSitterInfo struct {
	Externals []string
	Notes     []string
}

// FromTreeSitter builds a grammar from a tree-sitter grammar.json.
// token and pattern rules become matchers. Extras are skipped. Rules listed
// as inline are substituted into the productions that use them. A terminal
// supplied by an external scanner has no matcher.
func FromTreeSitter(data []byte) (*Grammar, *TreeSitterInfo, error) {
	file, err := decodeTreeSitter(data)
	if err != nil {
		return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: "+err.Error())
	}
	if len(file.rules) == 0 {
		return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: no rules")
	}
	b := &tsBld{
		g:        New(file.name),
		rules:    map[string]*tsNode{},
		order:    file.rules,
		inline:   map[string]bool{},
		reserved: map[string]bool{},
		info:     &TreeSitterInfo{},
	}
	for _, r := range file.rules {
		b.rules[r.name] = r.node
	}
	for _, name := range file.inline {
		b.inline[name] = true
	}
	b.ranks = precedenceRanks(file.precedences)
	if file.word != "" {
		b.word = file.word
	}
	for ctx, items := range file.reserved {
		if ctx != "global" {
			b.info.Notes = append(b.info.Notes, "reserved context "+ctx+" is not applied")
			continue
		}
		for _, it := range items {
			if s, ok := it.strVal(); ok {
				b.reserved[s] = true
			}
		}
	}
	for _, r := range file.rules {
		if isTokenRule(r.node) {
			if err := b.defineToken(r.name, r.node); err != nil {
				return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: "+err.Error())
			}
		}
	}
	for _, ext := range file.externals {
		b.external(ext)
	}
	for _, r := range file.rules {
		if isTokenRule(r.node) {
			continue
		}
		if err := b.defineSyntax(r.name, r.node); err != nil {
			return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: "+err.Error())
		}
	}
	b.applySymbolPrec()
	b.substituteInline()
	b.markKeywords()
	for _, extra := range file.extras {
		if err := b.skipExtra(extra); err != nil {
			return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: "+err.Error())
		}
	}
	if b.fields > 0 {
		b.info.Notes = append(b.info.Notes, fmt.Sprintf("%d fields are not stored on nodes", b.fields))
	}
	start := file.rules[0].name
	if b.g.syms[b.g.byName[start]].kind == kindTerm {
		wrap := b.g.Rule("source")
		wrap.Seq(b.g.Sym(start))
		start = "source"
	}
	b.g.Start(start)
	if len(b.g.syms) > int(^SymbolID(0)) {
		return nil, nil, errs.Wrap(errs.ErrInvalid, "tree-sitter: too many symbols")
	}
	b.info.Notes = dedupeStrings(b.info.Notes)
	return b.g, b.info, nil
}

func dedupeStrings(in []string) []string {
	if len(in) < 2 {
		return in
	}
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

type tsRule struct {
	name string
	node *tsNode
}

type tsFile struct {
	name        string
	word        string
	rules       []tsRule
	extras      []*tsNode
	externals   []*tsNode
	inline      []string
	precedences [][]*tsNode
	reserved    map[string][]*tsNode
}

type tsNode struct {
	Type        string          `json:"type"`
	Value       json.RawMessage `json:"value"`
	Name        string          `json:"name"`
	Named       bool            `json:"named"`
	Members     []*tsNode       `json:"members"`
	Content     *tsNode         `json:"content"`
	ContextName string          `json:"context_name"`
}

func (n *tsNode) strVal() (string, bool) {
	if n == nil || len(n.Value) == 0 {
		return "", false
	}
	var s string
	if json.Unmarshal(n.Value, &s) != nil {
		return "", false
	}
	return s, true
}

func (n *tsNode) intVal() (int, bool) {
	if n == nil || len(n.Value) == 0 {
		return 0, false
	}
	var i int
	if json.Unmarshal(n.Value, &i) != nil {
		return 0, false
	}
	return i, true
}

func decodeTreeSitter(data []byte) (*tsFile, error) {
	var raw struct {
		Name        string               `json:"name"`
		Word        json.RawMessage      `json:"word"`
		Rules       json.RawMessage      `json:"rules"`
		Extras      []*tsNode            `json:"extras"`
		Externals   []*tsNode            `json:"externals"`
		Inline      []string             `json:"inline"`
		Precedences [][]*tsNode          `json:"precedences"`
		Reserved    map[string][]*tsNode `json:"reserved"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rules, err := decodeRules(raw.Rules)
	if err != nil {
		return nil, err
	}
	word := ""
	if len(raw.Word) > 0 && string(raw.Word) != "null" {
		if json.Unmarshal(raw.Word, &word) != nil {
			var sym tsNode
			if err := json.Unmarshal(raw.Word, &sym); err != nil {
				return nil, errs.Wrap(errs.ErrInvalid, "word: "+err.Error(), err)
			}
			word = sym.Name
		}
	}
	return &tsFile{
		name:        raw.Name,
		word:        word,
		rules:       rules,
		extras:      raw.Extras,
		externals:   raw.Externals,
		inline:      raw.Inline,
		precedences: raw.Precedences,
		reserved:    raw.Reserved,
	}, nil
}

func decodeRules(raw json.RawMessage) ([]tsRule, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errs.Wrap(errs.ErrInvalid, "rules is not an object")
	}
	var rules []tsRule
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, ok := tok.(string)
		if !ok {
			return nil, errs.Wrap(errs.ErrInvalid, "rule name is not a string")
		}
		var node tsNode
		if err := dec.Decode(&node); err != nil {
			return nil, errs.Wrap(errs.ErrInvalid, fmt.Sprintf("rule %s: %s", name, err), err)
		}
		rules = append(rules, tsRule{name: name, node: &node})
	}
	return rules, nil
}

type tsAlt struct {
	syms   []*Symbol
	prec   int
	has    bool
	assoc  Assoc
	dyn    int
	hasDyn bool
}

type tsBld struct {
	g        *Grammar
	rules    map[string]*tsNode
	order    []tsRule
	inline   map[string]bool
	reserved map[string]bool
	ranks    map[string]int
	word     string
	fields   int
	info     *TreeSitterInfo
	visiting map[string]bool
	lits     []SymbolID
	byLit    map[string]SymbolID
	litText  map[SymbolID]string
}

func (b *tsBld) defineToken(name string, node *tsNode) error {
	body, imm, lexPrec, hasLex := unwrapToken(node)
	re, err := b.reOf(body)
	if err != nil {
		return errs.Wrap(errs.ErrInvalid, fmt.Sprintf("token %s: %s", name, err), err)
	}
	s := b.g.Token(name)
	s.Match(re.matchLen)
	b.g.syms[s.id].immediate = imm
	if hasLex {
		b.g.syms[s.id].lexPrec = lexPrec
	}
	return nil
}

func (b *tsBld) defineSyntax(name string, node *tsNode) error {
	rule := b.g.Rule(name)
	if b.inline[name] && !isHidden(name) {
		b.g.syms[rule.id].hidden = true
		b.g.syms[rule.id].named = false
	}
	alts, err := b.alts(node)
	if err != nil {
		return errs.Wrap(errs.ErrInvalid, name+": "+err.Error(), err)
	}
	if len(alts) == 0 {
		rule.Seq()
		return nil
	}
	for _, a := range alts {
		b.addAlt(rule, a)
	}
	return nil
}

func (b *tsBld) addAlt(rule *Rule, a tsAlt) {
	p := rule.Seq(a.syms...)
	if a.has {
		switch a.assoc {
		case AssocLeft:
			p.Left(a.prec)
		case AssocRight:
			p.Right(a.prec)
		default:
			p.Prec(a.prec)
		}
	}
	if a.hasDyn {
		p.Dynamic(a.dyn)
	}
}

func (b *tsBld) alts(n *tsNode) ([]tsAlt, error) {
	if n == nil {
		return []tsAlt{{}}, nil
	}
	switch n.Type {
	case "CHOICE":
		var out []tsAlt
		for _, m := range n.Members {
			part, err := b.alts(m)
			if err != nil {
				return nil, err
			}
			out = append(out, part...)
			if len(out) > 512 {
				s, err := b.group(n)
				return []tsAlt{{syms: []*Symbol{s}}}, err
			}
		}
		return out, nil
	case "SEQ":
		parts := []tsAlt{{}}
		for _, m := range n.Members {
			one, err := b.alts(m)
			if err != nil {
				return nil, err
			}
			var next []tsAlt
			for _, p := range parts {
				for _, a := range one {
					c := p
					c.syms = append(append([]*Symbol{}, p.syms...), a.syms...)
					if a.has && !c.has {
						c.prec, c.has, c.assoc = a.prec, true, a.assoc
					}
					if a.hasDyn {
						c.dyn += a.dyn
						c.hasDyn = true
					}
					next = append(next, c)
				}
			}
			if len(next) > 512 {
				s, err := b.group(n)
				return []tsAlt{{syms: []*Symbol{s}}}, err
			}
			parts = next
		}
		return parts, nil
	case "PREC", "PREC_LEFT", "PREC_RIGHT":
		inner, err := b.alts(n.Content)
		if err != nil {
			return nil, err
		}
		prec, has := b.precVal(n)
		assoc := AssocNone
		if n.Type == "PREC_LEFT" {
			assoc = AssocLeft
		} else if n.Type == "PREC_RIGHT" {
			assoc = AssocRight
		}
		for i := range inner {
			inner[i].prec = prec
			inner[i].has = has
			inner[i].assoc = assoc
		}
		return inner, nil
	case "PREC_DYNAMIC":
		inner, err := b.alts(n.Content)
		if err != nil {
			return nil, err
		}
		d, ok := n.intVal()
		if !ok {
			b.info.Notes = append(b.info.Notes, "dynamic precedence is not an integer")
			return inner, nil
		}
		for i := range inner {
			inner[i].dyn += d
			inner[i].hasDyn = true
		}
		return inner, nil
	case "BLANK":
		return []tsAlt{{}}, nil
	case "FIELD":
		b.fields++
		return b.alts(n.Content)
	case "RESERVED":
		if n.ContextName != "" && n.ContextName != "global" {
			b.info.Notes = append(b.info.Notes, "reserved context "+n.ContextName+" is not applied")
		}
		return b.alts(n.Content)
	default:
		s, err := b.atom(n)
		if err != nil {
			return nil, err
		}
		return []tsAlt{{syms: []*Symbol{s}}}, nil
	}
}

func (b *tsBld) atom(n *tsNode) (*Symbol, error) {
	switch n.Type {
	case "STRING":
		s, _ := n.strVal()
		return b.lit(s), nil
	case "PATTERN":
		return b.patternToken("", n)
	case "SYMBOL":
		return b.g.Sym(n.Name), nil
	case "TOKEN", "IMMEDIATE_TOKEN":
		return b.anonToken(n)
	case "ALIAS":
		return b.alias(n)
	case "FIELD":
		b.fields++
		return b.atom(n.Content)
	case "RESERVED":
		if n.ContextName != "" && n.ContextName != "global" {
			b.info.Notes = append(b.info.Notes, "reserved context "+n.ContextName+" is not applied")
		}
		return b.atom(n.Content)
	case "BLANK":
		r := b.g.hidden("$blank")
		r.Seq()
		return b.g.Sym(b.g.syms[r.id].name), nil
	case "REPEAT":
		inner, err := b.atom(n.Content)
		if err != nil {
			return nil, err
		}
		return b.g.Repeat(inner), nil
	case "REPEAT1":
		inner, err := b.atom(n.Content)
		if err != nil {
			return nil, err
		}
		return b.g.Repeat1(inner), nil
	default:
		return b.group(n)
	}
}

func (b *tsBld) group(n *tsNode) (*Symbol, error) {
	rule := b.g.hidden("$grp")
	alts, err := b.flatAlts(n)
	if err != nil {
		return nil, err
	}
	if len(alts) == 0 {
		rule.Seq()
	}
	for _, a := range alts {
		b.addAlt(rule, a)
	}
	return b.g.Sym(b.g.syms[rule.id].name), nil
}

// flatAlts lowers a node without calling group, so a choice cap cannot recurse.
func (b *tsBld) flatAlts(n *tsNode) ([]tsAlt, error) {
	switch n.Type {
	case "CHOICE":
		var out []tsAlt
		for _, m := range n.Members {
			part, err := b.flatAlts(m)
			if err != nil {
				return nil, err
			}
			out = append(out, part...)
		}
		return out, nil
	case "SEQ":
		parts := []tsAlt{{}}
		for _, m := range n.Members {
			one, err := b.flatAlts(m)
			if err != nil {
				return nil, err
			}
			var next []tsAlt
			for _, p := range parts {
				for _, a := range one {
					c := p
					c.syms = append(append([]*Symbol{}, p.syms...), a.syms...)
					if a.has {
						c.prec, c.has, c.assoc = a.prec, true, a.assoc
					}
					if a.hasDyn {
						c.dyn += a.dyn
						c.hasDyn = true
					}
					next = append(next, c)
				}
			}
			parts = next
		}
		return parts, nil
	default:
		return b.alts(n)
	}
}

func (b *tsBld) alias(n *tsNode) (*Symbol, error) {
	name, _ := n.strVal()
	if name == "" {
		name = "alias"
	}
	rname := name
	if _, ok := b.g.byName[rname]; ok {
		rname = fmt.Sprintf("%s#%d", name, len(b.g.syms))
	}
	rule := b.g.Rule(rname)
	if !n.Named {
		b.g.syms[rule.id].named = false
	}
	alts, err := b.alts(n.Content)
	if err != nil {
		return nil, err
	}
	for _, a := range alts {
		b.addAlt(rule, a)
	}
	return b.g.Sym(rname), nil
}

func (b *tsBld) anonToken(n *tsNode) (*Symbol, error) {
	body, imm, lexPrec, hasLex := unwrapToken(n)
	re, err := b.reOf(body)
	if err != nil {
		return nil, err
	}
	s := b.g.Token(fmt.Sprintf("$tok#%d", len(b.g.syms)))
	s.Match(re.matchLen)
	b.g.syms[s.id].immediate = imm
	if hasLex {
		b.g.syms[s.id].lexPrec = lexPrec
	}
	return s, nil
}

func (b *tsBld) patternToken(name string, n *tsNode) (*Symbol, error) {
	pat, _ := n.strVal()
	re, err := compilePattern(pat)
	if err != nil {
		return nil, errs.Wrap(errs.ErrSyntax, fmt.Sprintf("pattern %q: %s", pat, err), err)
	}
	if name == "" {
		name = fmt.Sprintf("$pat#%d", len(b.g.syms))
	}
	s := b.g.Token(name)
	s.Match(re.matchLen)
	return s, nil
}

func (b *tsBld) lit(text string) *Symbol {
	if b.byLit == nil {
		b.byLit = map[string]SymbolID{}
		b.litText = map[SymbolID]string{}
	}
	if id, ok := b.byLit[text]; ok {
		b.lits = append(b.lits, id)
		return &Symbol{g: b.g, id: id}
	}
	// A string token and a rule may share this spelling. Tree-sitter keeps
	// them as different symbols. The anonymous one takes a private name so
	// the rule can keep the spelling. Leaves still print the source text.
	var s *Symbol
	_, taken := b.g.byName[text]
	_, rule := b.rules[text]
	if taken || rule {
		s = b.g.Token(fmt.Sprintf("$q#%d", len(b.g.syms)))
		b.g.syms[s.id].named = false
		b.g.syms[s.id].match = hasBytes(text)
		b.g.syms[s.id].lexPrec = 1
	} else {
		s = b.g.Literal(text)
	}
	b.byLit[text] = s.id
	b.litText[s.id] = text
	b.lits = append(b.lits, s.id)
	return s
}

func (b *tsBld) external(n *tsNode) {
	switch n.Type {
	case "STRING":
		s, _ := n.strVal()
		b.lit(s)
		b.info.Externals = append(b.info.Externals, fmt.Sprintf("%q", s))
	case "SYMBOL":
		b.info.Externals = append(b.info.Externals, n.Name)
		if _, ok := b.rules[n.Name]; ok {
			return
		}
		if id, ok := b.g.byName[n.Name]; ok {
			if b.g.syms[id].kind == kindTerm && b.g.syms[id].match == nil {
				b.g.syms[id].external = true
			}
			return
		}
		s := b.g.Token(n.Name)
		b.g.syms[s.id].external = true
	default:
		b.info.Notes = append(b.info.Notes, "external "+n.Type+" is not a matcher")
	}
}

func (b *tsBld) skipExtra(n *tsNode) error {
	switch n.Type {
	case "PATTERN":
		s, err := b.patternToken("", n)
		if err != nil {
			return err
		}
		b.g.Skip(s)
		return nil
	case "STRING":
		text, _ := n.strVal()
		b.g.Skip(b.lit(text))
		return nil
	case "SYMBOL":
		id, ok := b.g.byName[n.Name]
		if !ok || b.g.syms[id].kind != kindTerm {
			b.info.Notes = append(b.info.Notes, "extra "+n.Name+" is not a token")
			return nil
		}
		if b.g.syms[id].match == nil {
			b.info.Notes = append(b.info.Notes, "extra "+n.Name+" comes from the external scanner")
			return nil
		}
		b.g.Skip(&Symbol{g: b.g, id: id})
		return nil
	default:
		b.info.Notes = append(b.info.Notes, "extra "+n.Type)
		return nil
	}
}

func (b *tsBld) reOf(n *tsNode) (*rex, error) {
	if n == nil {
		return reEmpty(), nil
	}
	switch n.Type {
	case "STRING":
		s, _ := n.strVal()
		return reLit(s), nil
	case "PATTERN":
		s, _ := n.strVal()
		return compilePattern(s)
	case "BLANK":
		return reEmpty(), nil
	case "SEQ":
		xs := make([]*rex, 0, len(n.Members))
		for _, m := range n.Members {
			x, err := b.reOf(m)
			if err != nil {
				return nil, err
			}
			xs = append(xs, x)
		}
		return reCat(xs...), nil
	case "CHOICE":
		xs := make([]*rex, 0, len(n.Members))
		for _, m := range n.Members {
			x, err := b.reOf(m)
			if err != nil {
				return nil, err
			}
			xs = append(xs, x)
		}
		return reAlt(xs...), nil
	case "REPEAT":
		x, err := b.reOf(n.Content)
		if err != nil {
			return nil, err
		}
		return reRep(x, 0, -1), nil
	case "REPEAT1":
		x, err := b.reOf(n.Content)
		if err != nil {
			return nil, err
		}
		return reRep(x, 1, -1), nil
	case "TOKEN", "IMMEDIATE_TOKEN", "PREC", "PREC_LEFT", "PREC_RIGHT", "PREC_DYNAMIC", "FIELD", "ALIAS", "RESERVED":
		return b.reOf(n.Content)
	case "SYMBOL":
		other, ok := b.rules[n.Name]
		if !ok {
			return nil, errs.Wrap(errs.ErrUnknown, "unknown symbol "+n.Name)
		}
		if b.visiting == nil {
			b.visiting = map[string]bool{}
		}
		if b.visiting[n.Name] {
			return nil, errs.Wrap(errs.ErrCycle, "cycle through "+n.Name)
		}
		b.visiting[n.Name] = true
		defer delete(b.visiting, n.Name)
		return b.reOf(other)
	default:
		return nil, errs.Wrap(errs.ErrUnsupported, "cannot lex "+n.Type)
	}
}

func (b *tsBld) precVal(n *tsNode) (int, bool) {
	if s, ok := n.strVal(); ok {
		r, ok := b.ranks["n:"+s]
		if !ok {
			b.info.Notes = append(b.info.Notes, "undeclared precedence "+s)
			return 0, true
		}
		return r, true
	}
	if i, ok := n.intVal(); ok {
		return i, true
	}
	b.info.Notes = append(b.info.Notes, "precedence is not a name or an integer")
	return 0, false
}

func (b *tsBld) applySymbolPrec() {
	for i, p := range b.g.prods {
		if p.hasPrec {
			continue
		}
		name := b.g.syms[p.lhs].name
		r, ok := b.ranks["s:"+name]
		if !ok {
			continue
		}
		b.g.prods[i].prec = r
		b.g.prods[i].hasPrec = true
	}
}

func (b *tsBld) markKeywords() {
	var word Match
	if b.word != "" {
		if id, ok := b.g.byName[b.word]; ok && b.g.syms[id].match != nil {
			word = b.g.syms[id].match
			if len(b.reserved) > 0 {
				inner := word
				reserved := b.reserved
				b.g.syms[id].match = func(src []byte, at int) int {
					n := inner(src, at)
					if n > 0 && reserved[string(src[at:at+n])] {
						return 0
					}
					return n
				}
				word = inner
			}
		}
	}
	if word == nil {
		return
	}
	seen := map[SymbolID]bool{}
	for _, id := range b.lits {
		if seen[id] {
			continue
		}
		seen[id] = true
		text := b.g.syms[id].name
		if t, ok := b.litText[id]; ok {
			text = t
		}
		if word([]byte(text), 0) == len(text) {
			b.g.syms[id].lexPrec = 2
		}
	}
}

func (b *tsBld) substituteInline() {
	ids := map[SymbolID]bool{}
	for name := range b.inline {
		id, ok := b.g.byName[name]
		if !ok || b.g.syms[id].kind == kindTerm {
			continue
		}
		ids[id] = true
	}
	if len(ids) == 0 {
		return
	}
	const maxProds = 12000
	for pass := 0; pass < 6; pass++ {
		changed := false
		next := make([]prodRec, 0, len(b.g.prods))
		for _, p := range b.g.prods {
			out := expandInline(p, ids, b.g.prods)
			if len(out) != 1 || !sameProd(out[0], p) {
				changed = true
			}
			if len(next)+len(out) > maxProds {
				b.info.Notes = append(b.info.Notes, "inline substitution stopped at the production cap")
				return
			}
			next = append(next, out...)
		}
		b.g.prods = next
		if !changed {
			break
		}
	}
	ref := map[SymbolID]bool{}
	for _, p := range b.g.prods {
		for _, s := range p.rhs {
			ref[s] = true
		}
	}
	kept := make([]prodRec, 0, len(b.g.prods))
	for _, p := range b.g.prods {
		if ids[p.lhs] && !ref[p.lhs] {
			continue
		}
		kept = append(kept, p)
	}
	b.g.prods = kept
}

func expandInline(p prodRec, ids map[SymbolID]bool, all []prodRec) []prodRec {
	if ids[p.lhs] {
		return []prodRec{p}
	}
	for i, sym := range p.rhs {
		if !ids[sym] {
			continue
		}
		var alts []prodRec
		recursive := false
		for _, q := range all {
			if q.lhs != sym {
				continue
			}
			for _, s := range q.rhs {
				if s == sym {
					recursive = true
				}
			}
			alts = append(alts, q)
		}
		if recursive || len(alts) == 0 {
			continue
		}
		out := make([]prodRec, 0, len(alts))
		for _, a := range alts {
			rhs := make([]SymbolID, 0, len(p.rhs)+len(a.rhs))
			rhs = append(rhs, p.rhs[:i]...)
			rhs = append(rhs, a.rhs...)
			rhs = append(rhs, p.rhs[i+1:]...)
			np := p
			np.rhs = rhs
			if !np.hasPrec && a.hasPrec {
				np.prec, np.hasPrec, np.assoc = a.prec, true, a.assoc
			}
			if a.hasDyn {
				np.dyn += a.dyn
				np.hasDyn = true
			}
			out = append(out, np)
		}
		return out
	}
	return []prodRec{p}
}

func sameProd(a, b prodRec) bool {
	if a.lhs != b.lhs || len(a.rhs) != len(b.rhs) {
		return false
	}
	for i := range a.rhs {
		if a.rhs[i] != b.rhs[i] {
			return false
		}
	}
	return true
}

func isTokenRule(n *tsNode) bool {
	seen := false
	for n != nil {
		switch n.Type {
		case "TOKEN", "IMMEDIATE_TOKEN":
			seen = true
			n = n.Content
		case "PREC", "PREC_LEFT", "PREC_RIGHT", "PREC_DYNAMIC", "FIELD":
			n = n.Content
		default:
			if seen {
				return true
			}
			return n.Type == "PATTERN" || n.Type == "STRING"
		}
	}
	return seen
}

func unwrapToken(n *tsNode) (body *tsNode, imm bool, lexPrec int, hasLex bool) {
	body = n
	for body != nil {
		switch body.Type {
		case "IMMEDIATE_TOKEN":
			imm = true
			body = body.Content
		case "TOKEN", "FIELD", "ALIAS", "RESERVED":
			body = body.Content
		case "PREC", "PREC_LEFT", "PREC_RIGHT":
			if i, ok := body.intVal(); ok {
				lexPrec, hasLex = i, true
			}
			body = body.Content
		default:
			return body, imm, lexPrec, hasLex
		}
	}
	return body, imm, lexPrec, hasLex
}

func precedenceRanks(lists [][]*tsNode) map[string]int {
	lower := map[string][]string{}
	var nodes []string
	seen := map[string]bool{}
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		nodes = append(nodes, s)
	}
	key := func(n *tsNode) string {
		if n.Type == "SYMBOL" {
			return "s:" + n.Name
		}
		if s, ok := n.strVal(); ok {
			return "n:" + s
		}
		return ""
	}
	for _, list := range lists {
		for i, a := range list {
			ka := key(a)
			add(ka)
			for _, b := range list[i+1:] {
				kb := key(b)
				add(kb)
				if ka != "" && kb != "" && ka != kb {
					lower[ka] = append(lower[ka], kb)
				}
			}
		}
	}
	memo := map[string]int{}
	vis := map[string]bool{}
	var rank func(string) int
	rank = func(n string) int {
		if v, ok := memo[n]; ok {
			return v
		}
		if vis[n] {
			return 0
		}
		vis[n] = true
		best := 0
		for _, m := range lower[n] {
			if r := rank(m) + 1; r > best {
				best = r
			}
		}
		vis[n] = false
		memo[n] = best
		return best
	}
	for _, n := range nodes {
		rank(n)
	}
	return memo
}
