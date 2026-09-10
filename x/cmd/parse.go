package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type fieldKind int

const (
	kindSwitch fieldKind = iota
	kindValue
	kindEither
	kindPositional
	kindRest
	kindRepeat
	kindCommand
)

type field struct {
	index []int
	kind  fieldKind
	long  string
	short rune
	cmd   string
}

type spec struct {
	root   reflect.Value
	fields []field
	longs  map[string]int
	shorts map[rune]int
	cmds   map[string]int
	pos    []int
}

func parseArgs(root reflect.Value, args []string) error {
	s, err := newSpec(root)
	if err != nil {
		return err
	}
	return s.parse(args)
}

func newSpec(root reflect.Value) (*spec, error) {
	if root.Kind() == reflect.Pointer {
		if root.IsNil() {
			if !root.CanSet() {
				return nil, fmt.Errorf("%w: nil args", ErrInvalidSpec)
			}
			root.Set(reflect.New(root.Type().Elem()))
		}
		root = root.Elem()
	}
	if root.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: args must be a struct", ErrInvalidSpec)
	}
	s := &spec{
		root:   root,
		longs:  make(map[string]int),
		shorts: make(map[rune]int),
		cmds:   make(map[string]int),
	}
	if err := s.addStruct(root, nil); err != nil {
		return nil, err
	}
	for i, idx := range s.pos {
		if s.fields[idx].kind == kindRest && i != len(s.pos)-1 {
			return nil, fmt.Errorf("%w: rest positional must be last", ErrInvalidSpec)
		}
	}
	if len(s.cmds) > 0 && len(s.pos) > 0 {
		return nil, fmt.Errorf("%w: command cannot mix with positionals", ErrInvalidSpec)
	}
	return s, nil
}

func (s *spec) addStruct(rv reflect.Value, prefix []int) error {
	t := rv.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		fv := rv.Field(i)
		if !fv.CanAddr() {
			continue
		}
		index := append(append([]int(nil), prefix...), i)
		if sf.Anonymous && shouldFlatten(fv) {
			ev, err := derefStruct(fv)
			if err != nil {
				return err
			}
			if err := s.addStruct(ev, index); err != nil {
				return err
			}
			continue
		}
		f, ok, err := newField(sf, fv)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		f.index = index
		if err := s.add(f); err != nil {
			return err
		}
	}
	return nil
}

func shouldFlatten(fv reflect.Value) bool {
	v := fv
	if v.Kind() == reflect.Pointer {
		if v.Type().Elem().Kind() != reflect.Struct {
			return false
		}
		if v.IsNil() {
			v = reflect.New(v.Type().Elem())
		}
	} else if v.Kind() != reflect.Struct {
		return false
	}
	rv := rvalue{v}
	return !rv.hasParse() && !rv.hasCount()
}

func derefStruct(fv reflect.Value) (reflect.Value, error) {
	if fv.Kind() != reflect.Pointer {
		return fv, nil
	}
	if fv.IsNil() {
		slot := rvalue{fv}.settable()
		slot.Set(reflect.New(fv.Type().Elem()))
		fv = slot
	}
	return fv.Elem(), nil
}

func newField(sf reflect.StructField, fv reflect.Value) (field, bool, error) {
	long := sf.Tag.Get("long")
	short, err := parseShort(sf.Tag.Get("short"))
	if err != nil {
		return field{}, false, err
	}
	tagged := long != "" || short != 0
	f := field{index: sf.Index, long: long, short: short}

	if fv.Kind() == reflect.Slice {
		if !(rvalue{reflect.New(fv.Type().Elem())}).hasParse() {
			return field{}, false, nil
		}
		if tagged {
			f.kind = kindRepeat
		} else {
			f.kind = kindRest
		}
		return f, true, nil
	}

	if cmd, ok := commandName(sf, fv); ok {
		f.kind = kindCommand
		f.cmd = cmd
		return f, true, nil
	}

	v := rvalue{fv}
	hasParse, hasCount := v.hasParse(), v.hasCount()
	if !hasParse && !hasCount {
		return field{}, false, nil
	}
	switch {
	case hasParse && hasCount:
		if !tagged {
			return field{}, false, fmt.Errorf("%w: %s needs short or long", ErrInvalidSpec, sf.Name)
		}
		f.kind = kindEither
	case hasCount:
		if !tagged {
			return field{}, false, fmt.Errorf("%w: %s needs short or long", ErrInvalidSpec, sf.Name)
		}
		f.kind = kindSwitch
	case tagged:
		f.kind = kindValue
	default:
		f.kind = kindPositional
	}
	return f, true, nil
}

func parseShort(tag string) (rune, error) {
	if tag == "" {
		return 0, nil
	}
	r, size := utf8.DecodeRuneInString(tag)
	if r == utf8.RuneError || size != len(tag) {
		return 0, fmt.Errorf("%w: short must be one character", ErrInvalidSpec)
	}
	return r, nil
}

func (s *spec) add(f field) error {
	idx := len(s.fields)
	if f.long != "" {
		if _, ok := s.longs[f.long]; ok {
			return fmt.Errorf("%w: duplicate long %s", ErrInvalidSpec, f.long)
		}
		s.longs[f.long] = idx
	}
	if f.short != 0 {
		if _, ok := s.shorts[f.short]; ok {
			return fmt.Errorf("%w: duplicate short %c", ErrInvalidSpec, f.short)
		}
		s.shorts[f.short] = idx
	}
	if f.kind == kindCommand {
		if _, ok := s.cmds[f.cmd]; ok {
			return fmt.Errorf("%w: duplicate command %s", ErrInvalidSpec, f.cmd)
		}
		s.cmds[f.cmd] = idx
	}
	if f.kind == kindPositional || f.kind == kindRest {
		if len(s.pos) > 0 && s.fields[s.pos[len(s.pos)-1]].kind == kindRest {
			return fmt.Errorf("%w: rest positional must be last", ErrInvalidSpec)
		}
		s.pos = append(s.pos, idx)
	}
	s.fields = append(s.fields, f)
	return nil
}

func (s *spec) parse(args []string) error {
	counts := make([]int, len(s.fields))
	posi := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			if len(s.cmds) > 0 {
				return s.feedCommand(args[i+1:], counts)
			}
			return s.feedPositionals(args[i+1:], &posi, counts)
		}
		if isOption(a) {
			if strings.HasPrefix(a, "--") {
				next, err := s.parseLong(a, args, i, counts)
				if err != nil {
					return err
				}
				i = next
				continue
			}
			next, err := s.parseShorts(a, args, i, counts)
			if err != nil {
				return err
			}
			i = next
			continue
		}
		if len(s.cmds) > 0 {
			return s.takeCommand(a, args[i+1:], counts)
		}
		if err := s.feedOne(a, &posi); err != nil {
			return err
		}
	}
	return s.applyCounts(counts)
}

func (s *spec) feedCommand(args []string, counts []int) error {
	if len(args) == 0 {
		return s.applyCounts(counts)
	}
	return s.takeCommand(args[0], args[1:], counts)
}

func (s *spec) takeCommand(name string, rest []string, counts []int) error {
	fi, ok := s.cmds[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, name)
	}
	if err := s.applyCounts(counts); err != nil {
		return err
	}
	fv := s.root.FieldByIndex(s.fields[fi].index)
	slot := rvalue{fv}.settable()
	if slot.Kind() != reflect.Pointer || slot.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%w: command must be a pointer to struct", ErrInvalidSpec)
	}
	if slot.IsNil() {
		slot.Set(reflect.New(slot.Type().Elem()))
	}
	return parseArgs(slot.Elem(), rest)
}

func commandName(sf reflect.StructField, fv reflect.Value) (string, bool) {
	if fv.Kind() != reflect.Pointer || fv.Type().Elem().Kind() != reflect.Struct {
		return "", false
	}
	elem := reflect.New(fv.Type().Elem())
	if (rvalue{elem}).hasParse() || (rvalue{elem}).hasCount() {
		return "", false
	}
	if name := sf.Tag.Get("cmd"); name != "" {
		return name, true
	}
	return strings.ToLower(sf.Name), true
}

func (s *spec) parseLong(a string, args []string, i int, counts []int) (int, error) {
	name, val, hasVal := strings.Cut(a[2:], "=")
	if name == "" {
		return i, fmt.Errorf("%w: %s", ErrUnknownFlag, a)
	}
	fi, err := s.lookupLong(name)
	if err != nil {
		return i, err
	}
	return s.applyOption(fi, val, hasVal, args, i, counts)
}

func (s *spec) parseShorts(a string, args []string, i int, counts []int) (int, error) {
	cluster := a[1:]
	for j := 0; j < len(cluster); {
		r, size := utf8.DecodeRuneInString(cluster[j:])
		fi, err := s.lookupShort(r)
		if err != nil {
			return i, err
		}
		f := s.fields[fi]
		rest := cluster[j+size:]
		switch f.kind {
		case kindSwitch:
			counts[fi]++
			j += size
		case kindEither:
			if rest != "" && rest[0] == '=' {
				if err := s.setValue(f, rest[1:]); err != nil {
					return i, err
				}
				return i, nil
			}
			counts[fi]++
			j += size
		case kindValue, kindRepeat:
			if rest != "" {
				if rest[0] == '=' {
					rest = rest[1:]
				}
				return i, s.setValue(f, rest)
			}
			if i+1 >= len(args) {
				return i, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
			}
			return i + 1, s.setValue(f, args[i+1])
		default:
			return i, fmt.Errorf("%w: %s is not a flag", ErrInvalidArgument, f.display())
		}
	}
	return i, nil
}

func (s *spec) applyOption(fi int, val string, hasVal bool, args []string, i int, counts []int) (int, error) {
	f := s.fields[fi]
	switch f.kind {
	case kindSwitch:
		if hasVal {
			return i, fmt.Errorf("%w: %s does not take a value", ErrInvalidArgument, f.display())
		}
		counts[fi]++
		return i, nil
	case kindEither:
		if hasVal {
			return i, s.setValue(f, val)
		}
		if v, ok := optionalCountValue(args, i); ok {
			return i + 1, s.setValue(f, v)
		}
		counts[fi]++
		return i, nil
	case kindValue, kindRepeat:
		if !hasVal {
			if i+1 >= len(args) {
				return i, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
			}
			return i + 1, s.setValue(f, args[i+1])
		}
		return i, s.setValue(f, val)
	default:
		return i, fmt.Errorf("%w: %s is not a flag", ErrInvalidArgument, f.display())
	}
}

func optionalCountValue(args []string, i int) (string, bool) {
	if i+1 >= len(args) || isOption(args[i+1]) {
		return "", false
	}
	if _, err := strconv.Atoi(args[i+1]); err != nil {
		return "", false
	}
	return args[i+1], true
}

func (s *spec) feedPositionals(args []string, posi *int, counts []int) error {
	for _, a := range args {
		if err := s.feedOne(a, posi); err != nil {
			return err
		}
	}
	return s.applyCounts(counts)
}

func (s *spec) feedOne(a string, posi *int) error {
	if *posi >= len(s.pos) {
		return fmt.Errorf("%w: unexpected argument %q", ErrInvalidArgument, a)
	}
	f := s.fields[s.pos[*posi]]
	if err := s.setValue(f, a); err != nil {
		return err
	}
	if f.kind != kindRest {
		*posi++
	}
	return nil
}

func (s *spec) applyCounts(counts []int) error {
	for i, n := range counts {
		if n == 0 {
			continue
		}
		if err := s.callCount(s.fields[i], n); err != nil {
			return err
		}
	}
	return nil
}

func (s *spec) setValue(f field, val string) error {
	v := rvalue{s.root.FieldByIndex(f.index)}
	var err error
	if f.kind == kindRest || f.kind == kindRepeat {
		err = v.appendParsed(val)
	} else {
		err = v.parse(val)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", f.display(), err)
	}
	return nil
}

func (s *spec) callCount(f field, n int) error {
	return rvalue{s.root.FieldByIndex(f.index)}.count(n)
}

func (s *spec) lookupLong(name string) (int, error) {
	i, ok := s.longs[name]
	if !ok {
		return 0, fmt.Errorf("%w: --%s", ErrUnknownFlag, name)
	}
	return i, nil
}

func (s *spec) lookupShort(r rune) (int, error) {
	i, ok := s.shorts[r]
	if !ok {
		return 0, fmt.Errorf("%w: -%c", ErrUnknownFlag, r)
	}
	return i, nil
}

func (f field) display() string {
	switch {
	case f.long != "":
		return "--" + f.long
	case f.short != 0:
		return "-" + string(f.short)
	default:
		return "argument"
	}
}

func isOption(a string) bool {
	return len(a) > 1 && a[0] == '-' && a != "-"
}

type rvalue struct{ reflect.Value }

func (v rvalue) ptr() rvalue {
	if v.Kind() == reflect.Pointer {
		return v
	}
	return rvalue{reflect.NewAt(v.Type(), v.Addr().UnsafePointer())}
}

func (v rvalue) settable() reflect.Value {
	return reflect.NewAt(v.Type(), v.Addr().UnsafePointer()).Elem()
}

func (v rvalue) hasParse() bool {
	return v.has("Parse", reflect.TypeFor[string]())
}

func (v rvalue) hasCount() bool {
	return v.has("Count", reflect.TypeFor[int]())
}

func (v rvalue) has(name string, in reflect.Type) bool {
	return methodSig(v.ptr().Value, name, in, reflect.TypeFor[error]())
}

func (v rvalue) parse(s string) error {
	return callErr(v.ptr().MethodByName("Parse"), reflect.ValueOf(s))
}

func (v rvalue) count(n int) error {
	return callErr(v.ptr().MethodByName("Count"), reflect.ValueOf(n))
}

func (v rvalue) appendParsed(val string) error {
	elem := reflect.New(v.Type().Elem())
	if err := (rvalue{elem}).parse(val); err != nil {
		return err
	}
	slice := v.ptr().Elem()
	slice.Set(reflect.Append(slice, elem.Elem()))
	return nil
}

func methodSig(ptr reflect.Value, name string, in, out reflect.Type) bool {
	m := ptr.MethodByName(name)
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	return t.NumIn() == 1 && t.In(0) == in && t.NumOut() == 1 && t.Out(0) == out
}

func callErr(m reflect.Value, arg reflect.Value) error {
	err, _ := m.Call([]reflect.Value{arg})[0].Interface().(error)
	return err
}
