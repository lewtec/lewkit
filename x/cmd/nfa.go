package cmd

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

type edgeKind byte

const (
	edgeEps edgeKind = iota
	edgeLit
	edgeFlag
	edgeVal
)

type edge struct {
	kind     edgeKind
	to       int
	lit      string
	long     string
	short    rune
	help     string
	choices  []string
	any      bool
	hasValue bool
	sk       SuggestKind
}

type nfa struct {
	start  int
	states [][]edge
}

type compiler struct {
	n *nfa
}

type flagInfo struct {
	f   field
	typ reflect.Type
}

func compileNFA(root reflect.Value) (*nfa, error) {
	s, err := newSpec(root)
	if err != nil {
		return nil, err
	}
	c := &compiler{n: &nfa{}}
	opt, _, err := c.spec(s, nil)
	if err != nil {
		return nil, err
	}
	c.n.start = opt
	return c.n, nil
}

func (c *compiler) state() int {
	c.n.states = append(c.n.states, nil)
	return len(c.n.states) - 1
}

func (c *compiler) add(from int, e edge) {
	c.n.states[from] = append(c.n.states[from], e)
}

func (c *compiler) eps(from, to int) {
	c.add(from, edge{kind: edgeEps, to: to})
}

func (c *compiler) lit(from int, tok string, to int, help string, sk SuggestKind) {
	c.add(from, edge{kind: edgeLit, to: to, lit: tok, help: help, sk: sk})
}

func (c *compiler) val(from, to int, any bool, choices []string, help string) {
	c.add(from, edge{kind: edgeVal, to: to, any: any, choices: choices, help: help, sk: SuggestValue})
}

func (c *compiler) flag(from int, f field, to int, hasValue bool) {
	c.add(from, edge{
		kind:     edgeFlag,
		to:       to,
		long:     f.long,
		short:    f.short,
		help:     f.help,
		hasValue: hasValue,
		sk:       SuggestFlag,
	})
}

func fieldType(s *spec, f field) reflect.Type {
	return s.root.Type().FieldByIndex(f.index).Type
}

func flagsOf(s *spec) []flagInfo {
	var out []flagInfo
	for _, f := range s.fields {
		switch f.kind {
		case kindSwitch, kindValue, kindEither, kindRepeat:
			t := fieldType(s, f)
			if f.kind == kindRepeat {
				t = t.Elem()
			}
			out = append(out, flagInfo{f: f, typ: t})
		}
	}
	return out
}

func mergeFlags(own, inherited []flagInfo) []flagInfo {
	out := slices.Clone(own)
	longs := make(map[string]struct{})
	shorts := make(map[rune]struct{})
	for _, f := range own {
		if f.f.long != "" {
			longs[f.f.long] = struct{}{}
		}
		if f.f.short != 0 {
			shorts[f.f.short] = struct{}{}
		}
	}
	for _, f := range inherited {
		ff := f
		if ff.f.long != "" {
			if _, ok := longs[ff.f.long]; ok {
				ff.f.long = ""
			}
		}
		if ff.f.short != 0 {
			if _, ok := shorts[ff.f.short]; ok {
				ff.f.short = 0
			}
		}
		if ff.f.long == "" && ff.f.short == 0 {
			continue
		}
		out = append(out, ff)
	}
	return out
}

func (c *compiler) spec(s *spec, inherited []flagInfo) (opt, all int, err error) {
	flags := mergeFlags(flagsOf(s), inherited)
	nPos := len(s.pos)
	hubs := make([]int, nPos+1)
	alls := make([]int, nPos+1)
	for i := range hubs {
		hubs[i] = c.state()
		alls[i] = c.state()
	}
	opt, all = hubs[0], alls[0]
	for i := range hubs {
		c.addFlags(hubs[i], flags)
		c.addEndOfOptions(s, i, hubs[i], alls)
	}
	for i, idx := range s.pos {
		c.pos(s, s.fields[idx], hubs[i], alls[i], hubs[i+1], alls[i+1])
	}
	if len(s.cmds) == 0 {
		return opt, all, nil
	}
	for _, name := range slices.Sorted(maps.Keys(s.cmds)) {
		f := s.fields[s.cmds[name]]
		childRoot := reflect.New(fieldType(s, f).Elem()).Elem()
		cs, err := newSpec(childRoot)
		if err != nil {
			return 0, 0, err
		}
		chOpt, chAll, err := c.spec(cs, flags)
		if err != nil {
			return 0, 0, err
		}
		help := firstLine(s.commandDescription(f))
		c.lit(opt, name, chOpt, help, SuggestCommand)
		c.lit(all, name, chAll, help, SuggestCommand)
	}
	return opt, all, nil
}

func (c *compiler) addFlags(hub int, flags []flagInfo) {
	for _, fi := range flags {
		switch fi.f.kind {
		case kindSwitch:
			c.flag(hub, fi.f, hub, false)
		case kindValue, kindRepeat:
			vs := c.state()
			c.flag(hub, fi.f, vs, true)
			c.consumeReq(vs, fi.typ, hub, true)
		case kindEither:
			c.flag(hub, fi.f, hub, false)
			vs := c.state()
			c.flag(hub, fi.f, vs, true)
			c.consumeReq(vs, fi.typ, hub, true)
		}
	}
}

func (c *compiler) addEndOfOptions(s *spec, posi int, from int, alls []int) {
	j := posi
	for j < len(s.pos) && s.fields[s.pos[j]].kind == kindRest && s.nextIsDash(j) {
		j++
	}
	if j < len(s.pos) && s.fields[s.pos[j]].kind == kindDash {
		c.lit(from, "--", alls[j+1], "", SuggestDash)
		return
	}
	c.lit(from, "--", alls[j], "", SuggestDash)
}

func (c *compiler) pos(s *spec, f field, fromOpt, fromAll, toOpt, toAll int) {
	t := fieldType(s, f)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch f.kind {
	case kindDash:
		c.lit(fromOpt, "--", toAll, "", SuggestDash)
		c.lit(fromAll, "--", toAll, "", SuggestDash)
		if f.optional {
			c.eps(fromOpt, toOpt)
			c.eps(fromAll, toAll)
		}
	case kindRest:
		elem := fieldType(s, f).Elem()
		c.val(fromOpt, fromOpt, false, choicesOf(elem), f.help)
		c.val(fromAll, fromAll, true, choicesOf(elem), f.help)
	default:
		if f.optional {
			c.eps(fromOpt, toOpt)
			c.eps(fromAll, toAll)
		}
		c.consumeReq(fromOpt, t, toOpt, false)
		c.consumeReq(fromAll, t, toAll, true)
	}
}

func typeHasParse(t reflect.Type) bool {
	return (rvalue{reflect.New(t)}).hasParse()
}

func (c *compiler) consumeReq(from int, t reflect.Type, dest int, any bool) {
	if t.Kind() == reflect.Pointer {
		c.consumeReq(from, t.Elem(), dest, any)
		return
	}
	if isDashType(t) {
		c.lit(from, "--", dest, "", SuggestDash)
		return
	}
	if typeHasParse(t) {
		c.val(from, dest, any, choicesOf(t), "")
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		c.product(from, t, dest, any)
	case reflect.Array:
		c.array(from, t, dest, any)
	case reflect.Slice:
		c.val(from, from, any, choicesOf(t.Elem()), "")
		c.eps(from, dest)
	}
}

func shouldFlattenType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		if t.Elem().Kind() != reflect.Struct {
			return false
		}
		t = t.Elem()
	} else if t.Kind() != reflect.Struct {
		return false
	}
	v := rvalue{reflect.New(t)}
	return !v.hasParse() && !v.hasCount()
}

func (c *compiler) consumeField(from int, t reflect.Type, dest int, any bool) {
	if t.Kind() == reflect.Pointer {
		c.eps(from, dest)
		c.consumeReq(from, t.Elem(), dest, any)
		return
	}
	c.consumeReq(from, t, dest, any)
}

func (c *compiler) product(from int, t reflect.Type, dest int, any bool) {
	var steps []reflect.Type
	var walk func(reflect.Type)
	walk = func(st reflect.Type) {
		if st.Kind() == reflect.Pointer {
			st = st.Elem()
		}
		for i := range st.NumField() {
			sf := st.Field(i)
			ft := sf.Type
			_, flatten := sf.Tag.Lookup("flatten")
			if (sf.Anonymous || flatten) && shouldFlattenType(ft) {
				walk(ft)
				continue
			}
			steps = append(steps, ft)
		}
	}
	walk(t)
	if len(steps) == 0 {
		c.eps(from, dest)
		return
	}
	cur := from
	for i, st := range steps {
		next := dest
		if i < len(steps)-1 {
			next = c.state()
		}
		c.consumeField(cur, st, next, any)
		cur = next
	}
}

func (c *compiler) array(from int, t reflect.Type, dest int, any bool) {
	n := t.Len()
	elem := t.Elem()
	cur := from
	for i := range n {
		next := dest
		if i < n-1 {
			next = c.state()
		}
		c.consumeReq(cur, elem, next, any)
		cur = next
	}
}

func (n *nfa) epsClose(set []int) []int {
	seen := make([]bool, len(n.states))
	var out []int
	var walk func(int)
	walk = func(s int) {
		if seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
		for _, e := range n.states[s] {
			if e.kind == edgeEps {
				walk(e.to)
			}
		}
	}
	for _, s := range set {
		walk(s)
	}
	return out
}

func uniqStates(in []int) []int {
	if len(in) < 2 {
		return in
	}
	slices.Sort(in)
	return slices.Compact(in)
}

func matchVal(e edge, tok string) bool {
	if tok == "--" {
		return false
	}
	if isOption(tok) && !e.any {
		return false
	}
	if len(e.choices) > 0 && !slices.Contains(e.choices, tok) {
		return false
	}
	return true
}

func isShortCluster(tok string) bool {
	return len(tok) > 2 && tok[0] == '-' && tok[1] != '-'
}

func (n *nfa) stepShort(set []int, cluster string) []int {
	if cluster == "" {
		return nil
	}
	r, size := utf8.DecodeRuneInString(cluster)
	rest := cluster[size:]
	var next []int
	for _, s := range set {
		for _, e := range n.states[s] {
			if e.kind != edgeFlag || e.short != r {
				continue
			}
			if rest == "" {
				next = append(next, e.to)
				continue
			}
			if e.hasValue {
				if rest[0] == '=' {
					rest = rest[1:]
				}
				next = append(next, n.step(n.epsClose([]int{e.to}), rest)...)
				continue
			}
			next = append(next, n.stepShort(n.epsClose([]int{e.to}), rest)...)
		}
	}
	return next
}

func (n *nfa) step(set []int, tok string) []int {
	set = n.epsClose(set)
	var next []int
	if name, val, ok := strings.Cut(tok, "="); ok && strings.HasPrefix(tok, "--") && name != "--" && name != "" {
		long := strings.TrimPrefix(name, "--")
		if long != "" {
			for _, s := range set {
				for _, e := range n.states[s] {
					if e.kind == edgeFlag && e.long == long && e.hasValue {
						next = append(next, n.step(n.epsClose([]int{e.to}), val)...)
					}
				}
			}
			return uniqStates(n.epsClose(next))
		}
	}
	for _, s := range set {
		for _, e := range n.states[s] {
			switch e.kind {
			case edgeLit:
				if tok == e.lit {
					next = append(next, e.to)
				}
			case edgeFlag:
				if e.long != "" && tok == "--"+e.long {
					next = append(next, e.to)
				}
				if e.short != 0 && tok == "-"+string(e.short) {
					next = append(next, e.to)
				}
			case edgeVal:
				if matchVal(e, tok) {
					next = append(next, e.to)
				}
			}
		}
	}
	if len(next) == 0 && isShortCluster(tok) {
		next = append(next, n.stepShort(set, tok[1:])...)
	}
	return uniqStates(n.epsClose(next))
}

func (n *nfa) suggest(set []int, prefix string) []Suggestion {
	set = n.epsClose(set)
	if long, val, ok := strings.Cut(strings.TrimPrefix(prefix, "--"), "="); ok && strings.HasPrefix(prefix, "--") && long != "" {
		return n.suggestFlagEq(set, long, val)
	}
	seen := make(map[string]struct{})
	var out []Suggestion
	add := func(s Suggestion) {
		if prefix != "" && !strings.HasPrefix(s.Text, prefix) {
			return
		}
		if _, ok := seen[s.Text]; ok {
			return
		}
		seen[s.Text] = struct{}{}
		out = append(out, s)
	}
	for _, s := range set {
		for _, e := range n.states[s] {
			switch e.kind {
			case edgeLit:
				add(Suggestion{Text: e.lit, Help: e.help, Kind: e.sk})
			case edgeFlag:
				if e.long != "" {
					add(Suggestion{Text: "--" + e.long, Help: e.help, Kind: SuggestFlag})
				}
				if e.short != 0 {
					add(Suggestion{Text: "-" + string(e.short), Help: e.help, Kind: SuggestFlag})
				}
			case edgeVal:
				for _, ch := range e.choices {
					add(Suggestion{Text: ch, Help: e.help, Kind: SuggestValue})
				}
			}
		}
	}
	slices.SortFunc(out, cmpSug)
	return out
}

func (n *nfa) suggestFlagEq(set []int, long, val string) []Suggestion {
	seen := make(map[string]struct{})
	var out []Suggestion
	for _, s := range set {
		for _, e := range n.states[s] {
			if e.kind != edgeFlag || e.long != long || !e.hasValue {
				continue
			}
			for _, sug := range n.suggest(n.epsClose([]int{e.to}), val) {
				text := "--" + long + "=" + sug.Text
				if _, ok := seen[text]; ok {
					continue
				}
				seen[text] = struct{}{}
				sug.Text = text
				out = append(out, sug)
			}
		}
	}
	slices.SortFunc(out, cmpSug)
	return out
}

func cmpSug(a, b Suggestion) int {
	if a.Kind != b.Kind {
		return int(a.Kind) - int(b.Kind)
	}
	return strings.Compare(a.Text, b.Text)
}
