package cmd

import (
	"maps"
	"reflect"
	"slices"

	"github.com/lewtec/lewkit/x/cmd/internal/ndfa"
)

type compiler struct {
	automaton *ndfa.NDFA
}

type flagInfo struct {
	field field
	typ   reflect.Type
}

func compileNDFA[T any]() (*ndfa.NDFA, error) {
	var zero T
	return compileNDFAValue(reflect.ValueOf(&zero).Elem())
}

func compileNDFAValue(root reflect.Value) (*ndfa.NDFA, error) {
	s, err := newSpec(root)
	if err != nil {
		return nil, err
	}
	c := &compiler{automaton: &ndfa.NDFA{}}
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

func (c *compiler) add(from int, e ndfa.Edge) {
	c.automaton.States[from] = append(c.automaton.States[from], e)
}

func (c *compiler) epsilon(from, to int) {
	c.add(from, ndfa.Edge{Kind: ndfa.EdgeEpsilon, To: to})
}

func (c *compiler) literal(from int, token string, to int, help string, suggestKind ndfa.SuggestKind) {
	c.add(from, ndfa.Edge{Kind: ndfa.EdgeLiteral, To: to, Literal: token, Help: help, SuggestKind: suggestKind})
}

func (c *compiler) value(from, to int, acceptAny bool, choices []string, help string) {
	c.add(from, ndfa.Edge{Kind: ndfa.EdgeValue, To: to, AcceptAny: acceptAny, Choices: choices, Help: help, SuggestKind: ndfa.SuggestValue})
}

func (c *compiler) flag(from int, f field, to int, hasValue bool) {
	c.add(from, ndfa.Edge{
		Kind:        ndfa.EdgeFlag,
		To:          to,
		Long:        f.long,
		Short:       f.short,
		Help:        f.help,
		HasValue:    hasValue,
		SuggestKind: ndfa.SuggestFlag,
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
		c.literal(optionStart, name, childOption, help, ndfa.SuggestCommand)
		c.literal(allPositionalStart, name, childAllPositional, help, ndfa.SuggestCommand)
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
		c.literal(from, "--", allPositionalHubs[j+1], "", ndfa.SuggestDash)
		return
	}
	c.literal(from, "--", allPositionalHubs[j], "", ndfa.SuggestDash)
}

func (c *compiler) positional(s *spec, f field, fromOption, fromAllPositional, toOption, toAllPositional int) {
	t := fieldType(s, f)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch f.kind {
	case kindDash:
		c.literal(fromOption, "--", toAllPositional, "", ndfa.SuggestDash)
		c.literal(fromAllPositional, "--", toAllPositional, "", ndfa.SuggestDash)
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
		c.literal(from, "--", dest, "", ndfa.SuggestDash)
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
