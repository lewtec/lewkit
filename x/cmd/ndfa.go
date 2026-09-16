package cmd

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

// EdgeKind is the label class on an NDFA transition.
type EdgeKind byte

const (
	EdgeEpsilon EdgeKind = iota
	EdgeLiteral
	EdgeFlag
	EdgeValue
)

// Edge is one NDFA transition.
type Edge struct {
	Kind        EdgeKind
	To          int
	Literal     string
	Long        string
	Short       rune
	Help        string
	Choices     []string
	AcceptAny   bool
	HasValue    bool
	SuggestKind SuggestKind
}

// NDFA is the token automaton compiled from a command spec.
type NDFA struct {
	Start  int
	States [][]Edge
}

type compiler struct {
	automaton *NDFA
}

type flagInfo struct {
	field field
	typ   reflect.Type
}

// CompileNDFA builds the token NDFA for T.
func CompileNDFA[T any]() (*NDFA, error) {
	var zero T
	return compileNDFA(reflect.ValueOf(&zero).Elem())
}

func compileNDFA(root reflect.Value) (*NDFA, error) {
	s, err := newSpec(root)
	if err != nil {
		return nil, err
	}
	c := &compiler{automaton: &NDFA{}}
	optionStart, _, err := c.spec(s, nil)
	if err != nil {
		return nil, err
	}
	c.automaton.Start = optionStart
	return c.automaton, nil
}

func (c *compiler) state() int {
	c.automaton.States = append(c.automaton.States, nil)
	return len(c.automaton.States) - 1
}

func (c *compiler) add(from int, e Edge) {
	c.automaton.States[from] = append(c.automaton.States[from], e)
}

func (c *compiler) epsilon(from, to int) {
	c.add(from, Edge{Kind: EdgeEpsilon, To: to})
}

func (c *compiler) literal(from int, token string, to int, help string, suggestKind SuggestKind) {
	c.add(from, Edge{Kind: EdgeLiteral, To: to, Literal: token, Help: help, SuggestKind: suggestKind})
}

func (c *compiler) value(from, to int, acceptAny bool, choices []string, help string) {
	c.add(from, Edge{Kind: EdgeValue, To: to, AcceptAny: acceptAny, Choices: choices, Help: help, SuggestKind: SuggestValue})
}

func (c *compiler) flag(from int, f field, to int, hasValue bool) {
	c.add(from, Edge{
		Kind:        EdgeFlag,
		To:          to,
		Long:        f.long,
		Short:       f.short,
		Help:        f.help,
		HasValue:    hasValue,
		SuggestKind: SuggestFlag,
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
			out = append(out, flagInfo{field: f, typ: t})
		}
	}
	return out
}

func mergeFlags(own, inherited []flagInfo) []flagInfo {
	out := slices.Clone(own)
	longs := make(map[string]struct{})
	shorts := make(map[rune]struct{})
	for _, info := range own {
		if info.field.long != "" {
			longs[info.field.long] = struct{}{}
		}
		if info.field.short != 0 {
			shorts[info.field.short] = struct{}{}
		}
	}
	for _, info := range inherited {
		next := info
		if next.field.long != "" {
			if _, ok := longs[next.field.long]; ok {
				next.field.long = ""
			}
		}
		if next.field.short != 0 {
			if _, ok := shorts[next.field.short]; ok {
				next.field.short = 0
			}
		}
		if next.field.long == "" && next.field.short == 0 {
			continue
		}
		out = append(out, next)
	}
	return out
}

func (c *compiler) spec(s *spec, inherited []flagInfo) (optionStart, allPositionalStart int, err error) {
	flags := mergeFlags(flagsOf(s), inherited)
	positionalCount := len(s.pos)
	optionHubs := make([]int, positionalCount+1)
	allPositionalHubs := make([]int, positionalCount+1)
	for i := range optionHubs {
		optionHubs[i] = c.state()
		allPositionalHubs[i] = c.state()
	}
	optionStart, allPositionalStart = optionHubs[0], allPositionalHubs[0]
	for i := range optionHubs {
		c.addFlags(optionHubs[i], flags)
		c.addEndOfOptions(s, i, optionHubs[i], allPositionalHubs)
	}
	for i, idx := range s.pos {
		c.positional(s, s.fields[idx], optionHubs[i], allPositionalHubs[i], optionHubs[i+1], allPositionalHubs[i+1])
	}
	if len(s.cmds) == 0 {
		return optionStart, allPositionalStart, nil
	}
	for _, name := range slices.Sorted(maps.Keys(s.cmds)) {
		f := s.fields[s.cmds[name]]
		childRoot := reflect.New(fieldType(s, f).Elem()).Elem()
		childSpec, err := newSpec(childRoot)
		if err != nil {
			return 0, 0, err
		}
		childOption, childAllPositional, err := c.spec(childSpec, flags)
		if err != nil {
			return 0, 0, err
		}
		help := firstLine(s.commandDescription(f))
		c.literal(optionStart, name, childOption, help, SuggestCommand)
		c.literal(allPositionalStart, name, childAllPositional, help, SuggestCommand)
	}
	return optionStart, allPositionalStart, nil
}

func (c *compiler) addFlags(hub int, flags []flagInfo) {
	for _, info := range flags {
		switch info.field.kind {
		case kindSwitch:
			c.flag(hub, info.field, hub, false)
		case kindValue, kindRepeat:
			valueStart := c.state()
			c.flag(hub, info.field, valueStart, true)
			c.consumeRequired(valueStart, info.typ, hub, true)
		case kindEither:
			c.flag(hub, info.field, hub, false)
			valueStart := c.state()
			c.flag(hub, info.field, valueStart, true)
			c.consumeRequired(valueStart, info.typ, hub, true)
		}
	}
}

func (c *compiler) addEndOfOptions(s *spec, positionalIndex int, from int, allPositionalHubs []int) {
	j := positionalIndex
	for j < len(s.pos) && s.fields[s.pos[j]].kind == kindRest && s.nextIsDash(j) {
		j++
	}
	if j < len(s.pos) && s.fields[s.pos[j]].kind == kindDash {
		c.literal(from, "--", allPositionalHubs[j+1], "", SuggestDash)
		return
	}
	c.literal(from, "--", allPositionalHubs[j], "", SuggestDash)
}

func (c *compiler) positional(s *spec, f field, fromOption, fromAllPositional, toOption, toAllPositional int) {
	t := fieldType(s, f)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch f.kind {
	case kindDash:
		c.literal(fromOption, "--", toAllPositional, "", SuggestDash)
		c.literal(fromAllPositional, "--", toAllPositional, "", SuggestDash)
		if f.optional {
			c.epsilon(fromOption, toOption)
			c.epsilon(fromAllPositional, toAllPositional)
		}
	case kindRest:
		elem := fieldType(s, f).Elem()
		c.value(fromOption, fromOption, false, choicesOf(elem), f.help)
		c.value(fromAllPositional, fromAllPositional, true, choicesOf(elem), f.help)
	default:
		if f.optional {
			c.epsilon(fromOption, toOption)
			c.epsilon(fromAllPositional, toAllPositional)
		}
		c.consumeRequired(fromOption, t, toOption, false)
		c.consumeRequired(fromAllPositional, t, toAllPositional, true)
	}
}

func typeHasParse(t reflect.Type) bool {
	return (rvalue{reflect.New(t)}).hasParse()
}

func (c *compiler) consumeRequired(from int, t reflect.Type, dest int, acceptAny bool) {
	if t.Kind() == reflect.Pointer {
		c.consumeRequired(from, t.Elem(), dest, acceptAny)
		return
	}
	if isDashType(t) {
		c.literal(from, "--", dest, "", SuggestDash)
		return
	}
	if typeHasParse(t) {
		c.value(from, dest, acceptAny, choicesOf(t), "")
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		c.product(from, t, dest, acceptAny)
	case reflect.Array:
		c.array(from, t, dest, acceptAny)
	case reflect.Slice:
		c.value(from, from, acceptAny, choicesOf(t.Elem()), "")
		c.epsilon(from, dest)
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

func (c *compiler) consumeField(from int, t reflect.Type, dest int, acceptAny bool) {
	if t.Kind() == reflect.Pointer {
		c.epsilon(from, dest)
		c.consumeRequired(from, t.Elem(), dest, acceptAny)
		return
	}
	c.consumeRequired(from, t, dest, acceptAny)
}

func (c *compiler) product(from int, t reflect.Type, dest int, acceptAny bool) {
	var steps []reflect.Type
	var walk func(reflect.Type)
	walk = func(structType reflect.Type) {
		if structType.Kind() == reflect.Pointer {
			structType = structType.Elem()
		}
		for i := range structType.NumField() {
			structField := structType.Field(i)
			memberType := structField.Type
			_, flatten := structField.Tag.Lookup("flatten")
			if (structField.Anonymous || flatten) && shouldFlattenType(memberType) {
				walk(memberType)
				continue
			}
			steps = append(steps, memberType)
		}
	}
	walk(t)
	if len(steps) == 0 {
		c.epsilon(from, dest)
		return
	}
	current := from
	for i, stepType := range steps {
		next := dest
		if i < len(steps)-1 {
			next = c.state()
		}
		c.consumeField(current, stepType, next, acceptAny)
		current = next
	}
}

func (c *compiler) array(from int, t reflect.Type, dest int, acceptAny bool) {
	length := t.Len()
	elem := t.Elem()
	current := from
	for i := range length {
		next := dest
		if i < length-1 {
			next = c.state()
		}
		c.consumeRequired(current, elem, next, acceptAny)
		current = next
	}
}

func (a *NDFA) epsilonClosure(set []int) []int {
	seen := make([]bool, len(a.States))
	var out []int
	var walk func(int)
	walk = func(state int) {
		if seen[state] {
			return
		}
		seen[state] = true
		out = append(out, state)
		for _, e := range a.States[state] {
			if e.Kind == EdgeEpsilon {
				walk(e.To)
			}
		}
	}
	for _, state := range set {
		walk(state)
	}
	return out
}

func uniqueStates(in []int) []int {
	if len(in) < 2 {
		return in
	}
	slices.Sort(in)
	return slices.Compact(in)
}

func matchValue(e Edge, token string) bool {
	if token == "--" {
		return false
	}
	if isOption(token) && !e.AcceptAny {
		return false
	}
	if len(e.Choices) > 0 && !slices.Contains(e.Choices, token) {
		return false
	}
	return true
}

func isShortCluster(token string) bool {
	return len(token) > 2 && token[0] == '-' && token[1] != '-'
}

func (a *NDFA) stepShort(set []int, cluster string) []int {
	if cluster == "" {
		return nil
	}
	r, size := utf8.DecodeRuneInString(cluster)
	rest := cluster[size:]
	var next []int
	for _, state := range set {
		for _, e := range a.States[state] {
			if e.Kind != EdgeFlag || e.Short != r {
				continue
			}
			if rest == "" {
				next = append(next, e.To)
				continue
			}
			if e.HasValue {
				if rest[0] == '=' {
					rest = rest[1:]
				}
				next = append(next, a.step(a.epsilonClosure([]int{e.To}), rest)...)
				continue
			}
			next = append(next, a.stepShort(a.epsilonClosure([]int{e.To}), rest)...)
		}
	}
	return next
}

func (a *NDFA) step(set []int, token string) []int {
	set = a.epsilonClosure(set)
	var next []int
	if name, value, ok := strings.Cut(token, "="); ok && strings.HasPrefix(token, "--") && name != "--" && name != "" {
		long := strings.TrimPrefix(name, "--")
		if long != "" {
			for _, state := range set {
				for _, e := range a.States[state] {
					if e.Kind == EdgeFlag && e.Long == long && e.HasValue {
						next = append(next, a.step(a.epsilonClosure([]int{e.To}), value)...)
					}
				}
			}
			return uniqueStates(a.epsilonClosure(next))
		}
	}
	for _, state := range set {
		for _, e := range a.States[state] {
			switch e.Kind {
			case EdgeLiteral:
				if token == e.Literal {
					next = append(next, e.To)
				}
			case EdgeFlag:
				if e.Long != "" && token == "--"+e.Long {
					next = append(next, e.To)
				}
				if e.Short != 0 && token == "-"+string(e.Short) {
					next = append(next, e.To)
				}
			case EdgeValue:
				if matchValue(e, token) {
					next = append(next, e.To)
				}
			}
		}
	}
	if len(next) == 0 && isShortCluster(token) {
		next = append(next, a.stepShort(set, token[1:])...)
	}
	return uniqueStates(a.epsilonClosure(next))
}

func (a *NDFA) suggest(set []int, prefix string) []Suggestion {
	set = a.epsilonClosure(set)
	if long, value, ok := strings.Cut(strings.TrimPrefix(prefix, "--"), "="); ok && strings.HasPrefix(prefix, "--") && long != "" {
		return a.suggestFlagEquals(set, long, value)
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
	for _, state := range set {
		for _, e := range a.States[state] {
			switch e.Kind {
			case EdgeLiteral:
				add(Suggestion{Text: e.Literal, Help: e.Help, Kind: e.SuggestKind})
			case EdgeFlag:
				if e.Long != "" {
					add(Suggestion{Text: "--" + e.Long, Help: e.Help, Kind: SuggestFlag})
				}
				if e.Short != 0 {
					add(Suggestion{Text: "-" + string(e.Short), Help: e.Help, Kind: SuggestFlag})
				}
			case EdgeValue:
				for _, choice := range e.Choices {
					add(Suggestion{Text: choice, Help: e.Help, Kind: SuggestValue})
				}
			}
		}
	}
	slices.SortFunc(out, compareSuggestion)
	return out
}

func (a *NDFA) suggestFlagEquals(set []int, long, value string) []Suggestion {
	seen := make(map[string]struct{})
	var out []Suggestion
	for _, state := range set {
		for _, e := range a.States[state] {
			if e.Kind != EdgeFlag || e.Long != long || !e.HasValue {
				continue
			}
			for _, suggestion := range a.suggest(a.epsilonClosure([]int{e.To}), value) {
				text := "--" + long + "=" + suggestion.Text
				if _, ok := seen[text]; ok {
					continue
				}
				seen[text] = struct{}{}
				suggestion.Text = text
				out = append(out, suggestion)
			}
		}
	}
	slices.SortFunc(out, compareSuggestion)
	return out
}

func compareSuggestion(a, b Suggestion) int {
	if a.Kind != b.Kind {
		return int(a.Kind) - int(b.Kind)
	}
	return strings.Compare(a.Text, b.Text)
}
