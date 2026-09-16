package cmd

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// Usage is the help text for T's flags, commands, and positionals.
func Usage[T any](name string) (string, error) {
	var zero T
	return usageOf(reflect.ValueOf(zero), name)
}

func usageOf(v reflect.Value, name string) (string, error) {
	return usageOfFlags(v, name, nil)
}

func usageSelected(root reflect.Value, name string) (string, error) {
	leaf, fullName, types := commandPath(root, name)
	var inherited []usageFlag
	for i := len(types) - 2; i >= 0; i-- {
		ancestor, err := specOfType(types[i])
		if err != nil {
			return "", err
		}
		inherited = mergeUsageFlags(inherited, usageFlagsOf(ancestor))
	}
	return usageOfFlags(leaf, fullName, inherited)
}

func specOfType(t reflect.Type) (*spec, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return newSpec(reflect.New(t).Elem())
}

func usageOfFlags(v reflect.Value, name string, inherited []usageFlag) (string, error) {
	t := v.Type()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	zero := reflect.New(t).Elem()
	s, err := newSpec(zero)
	if err != nil {
		return "", err
	}
	return s.usage(name, descriptionFrom(zero), inherited), nil
}

type usageFlag struct {
	field field
	help  string
}

func isUsageFlag(f field) bool {
	switch f.kind {
	case kindSwitch, kindValue, kindEither, kindRepeat:
		return true
	default:
		return false
	}
}

func usageFlagsOf(s *spec) []usageFlag {
	var out []usageFlag
	for _, f := range s.fields {
		if !isUsageFlag(f) {
			continue
		}
		out = append(out, usageFlag{field: f, help: s.lineHelp(f)})
	}
	return out
}

func mergeUsageFlags(own, inherited []usageFlag) []usageFlag {
	out := slices.Clone(own)
	longs := make(map[string]struct{})
	shorts := make(map[rune]struct{})
	for _, item := range own {
		if item.field.long != "" {
			longs[item.field.long] = struct{}{}
		}
		if item.field.short != 0 {
			shorts[item.field.short] = struct{}{}
		}
	}
	for _, item := range inherited {
		next := item
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
		if next.field.long != "" {
			longs[next.field.long] = struct{}{}
		}
		if next.field.short != 0 {
			shorts[next.field.short] = struct{}{}
		}
		out = append(out, next)
	}
	return out
}

func descriptionOf[T any]() string {
	var zero T
	return descriptionFrom(reflect.ValueOf(zero))
}

func descriptionFrom(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem())
			continue
		}
		v = v.Elem()
	}
	ptr := reflect.New(v.Type())
	if v.CanInterface() {
		ptr.Elem().Set(v)
	}
	if d, ok := ptr.Interface().(Describer); ok {
		return d.Description()
	}
	if d, ok := ptr.Elem().Interface().(Describer); ok {
		return d.Description()
	}
	return ""
}

func (s *spec) usage(name, desc string, inherited []usageFlag) string {
	var b strings.Builder
	if desc != "" {
		b.WriteString(desc)
		if !strings.HasSuffix(desc, "\n") {
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "Usage:\n  %s [flags]", name)
	if len(s.cmds) > 0 {
		fmt.Fprintf(&b, "\n  %s <command> [args]", name)
	}
	if len(s.pos) > 0 {
		fmt.Fprintf(&b, "\n  %s [flags]", name)
		for _, i := range s.pos {
			f := s.fields[i]
			switch f.kind {
			case kindRest:
				fmt.Fprintf(&b, " [%s...]", positionalName(f))
			case kindDash:
				b.WriteString(" --")
			default:
				fmt.Fprintf(&b, " <%s>", positionalName(f))
			}
		}
	}
	b.WriteByte('\n')

	var cmds, pos []field
	for _, f := range s.fields {
		switch f.kind {
		case kindCommand:
			cmds = append(cmds, f)
		case kindPositional, kindRest, kindProduct, kindArray, kindDash:
			pos = append(pos, f)
		}
	}
	s.writeGroup(&b, "Commands", cmds)
	writeUsageFlags(&b, mergeUsageFlags(usageFlagsOf(s), inherited))
	s.writeGroup(&b, "Arguments", pos)
	return b.String()
}

func writeUsageFlags(b *strings.Builder, flags []usageFlag) {
	if len(flags) == 0 {
		return
	}
	width := 0
	labels := make([]string, len(flags))
	for i, item := range flags {
		labels[i] = usageLabel(item.field)
		if n := len(labels[i]); n > width {
			width = n
		}
	}
	fmt.Fprintf(b, "\nFlags:\n")
	for i, item := range flags {
		fmt.Fprintf(b, "  %-*s  %s\n", width, labels[i], item.help)
	}
}

func (s *spec) writeGroup(b *strings.Builder, title string, fields []field) {
	if len(fields) == 0 {
		return
	}
	width := 0
	labels := make([]string, len(fields))
	for i, f := range fields {
		labels[i] = usageLabel(f)
		if n := len(labels[i]); n > width {
			width = n
		}
	}
	fmt.Fprintf(b, "\n%s:\n", title)
	for i, f := range fields {
		fmt.Fprintf(b, "  %-*s  %s\n", width, labels[i], s.lineHelp(f))
	}
}

func (s *spec) lineHelp(f field) string {
	help := f.help
	if help == "" && f.kind == kindCommand {
		help = firstLine(s.commandDescription(f))
	}
	var extra []string
	if names := envNames(f.env); len(names) > 0 {
		extra = append(extra, "env: "+strings.Join(names, ", "))
	}
	if names := choicesOf(s.root.FieldByIndex(f.index).Type()); len(names) > 0 {
		extra = append(extra, "choices: "+strings.Join(names, ", "))
	}
	if f.hasDef && f.def != "" {
		extra = append(extra, "default: "+f.def)
	}
	if len(extra) == 0 {
		return help
	}
	suffix := "(" + strings.Join(extra, ", ") + ")"
	if help == "" {
		return suffix
	}
	return help + " " + suffix
}

func (s *spec) commandDescription(f field) string {
	t := s.root.FieldByIndex(f.index).Type()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	ptr := reflect.New(t)
	if d, ok := ptr.Interface().(Describer); ok {
		return d.Description()
	}
	if d, ok := ptr.Elem().Interface().(Describer); ok {
		return d.Description()
	}
	return ""
}

func choicesOf(t reflect.Type) []string {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	v := reflect.New(t)
	if c, ok := v.Interface().(ArgChooser); ok {
		return c.ArgChoices()
	}
	if c, ok := v.Elem().Interface().(ArgChooser); ok {
		return c.ArgChoices()
	}
	return nil
}

func firstLine(s string) string {
	line, _, found := strings.Cut(s, "\n")
	if !found {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(line)
}

func usageLabel(f field) string {
	switch f.kind {
	case kindCommand:
		return f.cmd
	case kindPositional, kindRest, kindProduct, kindArray:
		return positionalName(f)
	case kindDash:
		return "--"
	default:
		return flagLabel(f)
	}
}

func flagLabel(f field) string {
	switch {
	case f.short != 0 && f.long != "":
		return fmt.Sprintf("-%c, --%s", f.short, f.long)
	case f.long != "":
		return "--" + f.long
	default:
		return "-" + string(f.short)
	}
}

func positionalName(f field) string {
	if f.long != "" {
		return f.long
	}
	return "arg"
}
