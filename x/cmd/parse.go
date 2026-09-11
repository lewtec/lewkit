package cmd

import (
	"fmt"
	"os"
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
	kindProduct
	kindArray
	kindDash
)

type field struct {
	index    []int
	kind     fieldKind
	long     string
	short    rune
	cmd      string
	help     string
	def      string
	hasDef   bool
	env      string
	optional bool
	maybePos bool
}

type spec struct {
	root   reflect.Value
	fields []field
	longs  map[string]int
	shorts map[rune]int
	cmds   map[string]int
	pos    []int
	set    []bool
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
	if err := s.finalize(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *spec) finalize() error {
	hasPos := false
	for _, f := range s.fields {
		if isPosKind(f.kind) {
			hasPos = true
			break
		}
	}
	if hasPos {
		for i := range s.fields {
			f := &s.fields[i]
			if !f.maybePos {
				continue
			}
			delete(s.cmds, f.cmd)
			f.kind = kindProduct
			f.optional = true
			f.cmd = ""
			f.maybePos = false
		}
	}
	s.pos = s.pos[:0]
	for i, f := range s.fields {
		if isPosKind(f.kind) {
			s.pos = append(s.pos, i)
		}
	}
	if err := s.checkPos(); err != nil {
		return err
	}
	if len(s.cmds) > 0 && len(s.pos) > 0 {
		return fmt.Errorf("%w: command cannot mix with positionals", ErrInvalidSpec)
	}
	return nil
}

func (s *spec) checkPos() error {
	greedy := false
	for _, idx := range s.pos {
		switch s.fields[idx].kind {
		case kindRest:
			greedy = true
		case kindDash:
			greedy = false
		default:
			if greedy {
				return fmt.Errorf("%w: rest positional must be last", ErrInvalidSpec)
			}
		}
	}
	return nil
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
		_, flatten := sf.Tag.Lookup("flatten")
		if (sf.Anonymous || flatten) && shouldFlatten(fv) {
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
		attachTypeDefault(&f, fv.Type())
		if err := s.add(f); err != nil {
			return err
		}
	}
	return nil
}

func attachTypeDefault(f *field, t reflect.Type) {
	if f.hasDef || !allowsDefault(f.kind) {
		return
	}
	d, ok := defaultOf(t)
	if !ok {
		return
	}
	f.def = d
	f.hasDef = true
}

func allowsDefault(k fieldKind) bool {
	switch k {
	case kindSwitch, kindValue, kindEither, kindPositional:
		return true
	default:
		return false
	}
}

func rejectDefaultOrEnv(sf reflect.StructField, f field) error {
	if f.hasDef {
		return fmt.Errorf("%w: default not allowed on %s", ErrInvalidSpec, sf.Name)
	}
	if f.env != "" {
		return fmt.Errorf("%w: env not allowed on %s", ErrInvalidSpec, sf.Name)
	}
	return nil
}

func defaultOf(t reflect.Type) (string, bool) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	v := rvalue{reflect.New(t)}
	if !v.hasArgDefault() {
		return "", false
	}
	return v.defaultString(), true
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
	f := field{index: sf.Index, long: long, short: short, help: sf.Tag.Get("help"), env: sf.Tag.Get("env")}
	if d, ok := sf.Tag.Lookup("default"); ok {
		f.def = d
		f.hasDef = true
	}

	if isDashType(fv.Type()) {
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		f.kind = kindDash
		return f, true, nil
	}
	if fv.Kind() == reflect.Pointer && isDashType(fv.Type().Elem()) {
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		f.kind = kindDash
		f.optional = true
		return f, true, nil
	}

	if fv.Kind() == reflect.Slice {
		if !isConsumable(fv.Type().Elem()) {
			return field{}, false, nil
		}
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		if tagged {
			f.kind = kindRepeat
		} else {
			f.kind = kindRest
		}
		return f, true, nil
	}

	if fv.Kind() == reflect.Array {
		if !isConsumable(fv.Type().Elem()) {
			return field{}, false, nil
		}
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		f.kind = kindArray
		return f, true, nil
	}

	if cmd, ok := commandName(sf, fv); ok {
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		f.kind = kindCommand
		f.cmd = cmd
		if sf.Tag.Get("cmd") == "" && !structHasCommandBits(fv.Type().Elem()) && isProduct(fv.Type().Elem()) {
			f.maybePos = true
		}
		return f, true, nil
	}

	if fv.Kind() == reflect.Struct && isProduct(fv.Type()) {
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		f.kind = kindProduct
		return f, true, nil
	}

	if fv.Kind() == reflect.Pointer && !tagged && isConsumable(fv.Type().Elem()) {
		if err := rejectDefaultOrEnv(sf, f); err != nil {
			return field{}, false, err
		}
		switch {
		case isProduct(fv.Type().Elem()):
			f.kind = kindProduct
		case fv.Type().Elem().Kind() == reflect.Array:
			f.kind = kindArray
		default:
			f.kind = kindPositional
		}
		f.optional = true
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
	if isPosKind(f.kind) {
		s.pos = append(s.pos, idx)
	}
	s.fields = append(s.fields, f)
	return nil
}

func (s *spec) parse(args []string) error {
	s.set = make([]bool, len(s.fields))
	counts := make([]int, len(s.fields))
	posi := 0
	allPos := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !allPos && a == "--" {
			if len(s.cmds) > 0 {
				return s.feedCommand(args[i+1:], counts)
			}
			for posi < len(s.pos) && s.fields[s.pos[posi]].kind == kindRest && s.nextIsDash(posi) {
				posi++
			}
			if posi < len(s.pos) && s.fields[s.pos[posi]].kind == kindDash {
				n, err := s.takePos(posi, args[i:], consumeMode{allPos: true, optional: s.fields[s.pos[posi]].optional})
				if err != nil {
					return err
				}
				if n == 0 {
					return fmt.Errorf("%w: expected --", ErrMissingValue)
				}
				i += n - 1
				posi++
				allPos = true
				continue
			}
			allPos = true
			continue
		}
		if !allPos && isOption(a) {
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
		if posi >= len(s.pos) {
			return fmt.Errorf("%w: unexpected argument %q", ErrInvalidArgument, a)
		}
		mode := consumeMode{allPos: allPos, stopDash: s.nextIsDash(posi), optional: s.fields[s.pos[posi]].optional}
		n, err := s.takePos(posi, args[i:], mode)
		if err != nil {
			return err
		}
		if n == 0 {
			posi++
			i--
			continue
		}
		i += n - 1
		posi++
	}
	return s.finish(counts)
}

func (s *spec) nextIsDash(posi int) bool {
	if posi+1 >= len(s.pos) {
		return false
	}
	switch s.fields[s.pos[posi+1]].kind {
	case kindDash, kindRest:
		return true
	default:
		return false
	}
}

func (s *spec) takePos(posi int, args []string, mode consumeMode) (int, error) {
	fi := s.pos[posi]
	f := s.fields[fi]
	fv := rvalue{s.root.FieldByIndex(f.index)}.settable()
	n, err := consumeValue(fv, args, mode)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", f.display(), err)
	}
	if n > 0 {
		s.mark(fi)
		return n, nil
	}
	if f.kind == kindRest || f.optional {
		return 0, nil
	}
	if f.kind == kindPositional {
		return 0, nil
	}
	return 0, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
}

func (s *spec) feedCommand(args []string, counts []int) error {
	if len(args) == 0 {
		return s.finish(counts)
	}
	return s.takeCommand(args[0], args[1:], counts)
}

func (s *spec) takeCommand(name string, rest []string, counts []int) error {
	fi, ok := s.cmds[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, name)
	}
	if err := s.finish(counts); err != nil {
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
				if err := s.setValueAt(fi, rest[1:]); err != nil {
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
				return s.takeFlagTokens(fi, []string{rest}, args, i, 0)
			}
			return s.takeFlagTokens(fi, nil, args, i, 1)
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
			return i, s.setValueAt(fi, val)
		}
		if v, ok := optionalCountValue(args, i); ok {
			return i + 1, s.setValueAt(fi, v)
		}
		counts[fi]++
		return i, nil
	case kindValue, kindRepeat:
		if hasVal {
			return s.takeFlagTokens(fi, []string{val}, args, i, 0)
		}
		return s.takeFlagTokens(fi, nil, args, i, 1)
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

func (s *spec) takeFlagTokens(fi int, prefix []string, args []string, i int, fromNext int) (int, error) {
	f := s.fields[fi]
	var tokens []string
	tokens = append(tokens, prefix...)
	if fromNext > 0 {
		if i+fromNext > len(args) {
			return i, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
		}
		tokens = append(tokens, args[i+fromNext:]...)
	}
	if len(tokens) == 0 {
		return i, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
	}
	n, err := s.consumeFlagValue(fi, tokens)
	if err != nil {
		return i, err
	}
	if n == 0 {
		return i, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
	}
	used := n - len(prefix)
	if used < 0 {
		used = 0
	}
	return i + used, nil
}

func (s *spec) consumeFlagValue(fi int, tokens []string) (int, error) {
	f := s.fields[fi]
	fv := rvalue{s.root.FieldByIndex(f.index)}.settable()
	if f.kind == kindRepeat {
		elem := reflect.New(fv.Type().Elem()).Elem()
		n, err := consumeValue(elem, tokens, consumeMode{allPos: true})
		if err != nil {
			return 0, fmt.Errorf("%s: %w", f.display(), err)
		}
		if n == 0 {
			return 0, fmt.Errorf("%w: %s", ErrMissingValue, f.display())
		}
		fv.Set(reflect.Append(fv, elem))
		s.mark(fi)
		return n, nil
	}
	n, err := consumeValue(fv, tokens, consumeMode{allPos: true})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", f.display(), err)
	}
	if n > 0 {
		s.mark(fi)
	}
	return n, nil
}

func (s *spec) finish(counts []int) error {
	if err := s.applyCounts(counts); err != nil {
		return err
	}
	if err := s.applyEnv(); err != nil {
		return err
	}
	if err := s.applyDefaults(); err != nil {
		return err
	}
	if s.skipRequired() {
		return nil
	}
	for i, f := range s.fields {
		if s.set[i] || f.hasDef || f.optional || !requiresPresence(f) {
			continue
		}
		return fmt.Errorf("%w: %s", ErrMissingValue, f.display())
	}
	return nil
}

func (s *spec) skipRequired() bool {
	for i, f := range s.fields {
		if s.set[i] && (f.long == "help" || f.long == "version") {
			return true
		}
	}
	return false
}

func requiresPresence(f field) bool {
	switch f.kind {
	case kindSwitch, kindValue, kindEither:
		return true
	case kindRest, kindPositional, kindRepeat, kindCommand:
		return false
	default:
		return isPosKind(f.kind)
	}
}

func (s *spec) applyCounts(counts []int) error {
	for i, n := range counts {
		if n == 0 {
			continue
		}
		if err := s.callCount(s.fields[i], n); err != nil {
			return err
		}
		s.mark(i)
	}
	return nil
}

func (s *spec) applyEnv() error {
	for i, f := range s.fields {
		if f.env == "" || s.set[i] {
			continue
		}
		val, ok := os.LookupEnv(f.env)
		if !ok {
			continue
		}
		if err := s.applyLiteral(f, val); err != nil {
			return fmt.Errorf("env %s: %w", f.env, err)
		}
		s.mark(i)
	}
	return nil
}

func (s *spec) applyDefaults() error {
	for i, f := range s.fields {
		if !f.hasDef || s.set[i] {
			continue
		}
		if err := s.applyDefault(f); err != nil {
			return err
		}
	}
	return nil
}

func (s *spec) applyDefault(f field) error {
	if err := s.applyLiteral(f, f.def); err != nil {
		return fmt.Errorf("default %w", err)
	}
	return nil
}

func (s *spec) applyLiteral(f field, val string) error {
	if f.kind == kindSwitch {
		on, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("%s: %w", f.display(), ErrInvalidArgument)
		}
		if !on {
			return nil
		}
		if err := s.callCount(f, 1); err != nil {
			return fmt.Errorf("%s: %w", f.display(), err)
		}
		return nil
	}
	return s.setValue(f, val)
}

func (s *spec) mark(fi int) {
	if s.set != nil {
		s.set[fi] = true
	}
}

func (s *spec) setValueAt(fi int, val string) error {
	if err := s.setValue(s.fields[fi], val); err != nil {
		return err
	}
	s.mark(fi)
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

func (v rvalue) hasArgDefault() bool {
	m := v.ptr().MethodByName("ArgDefault")
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	return t.NumIn() == 0 && t.NumOut() == 1 && t.Out(0) == reflect.TypeFor[string]()
}

func (v rvalue) defaultString() string {
	return v.ptr().MethodByName("ArgDefault").Call(nil)[0].String()
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
