package cmd

import (
	"fmt"
	"reflect"
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
	return mergeNamedFlags(own, inherited,
		func(item usageFlag) (string, rune) { return item.field.long, item.field.short },
		func(item usageFlag, long string, short rune) usageFlag {
			item.field.long = long
			item.field.short = short
			return item
		},
	)
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
	rows := make([]usageRow, len(flags))
	for i, item := range flags {
		rows[i] = usageRow{label: usageLabel(item.field), help: item.help}
	}
	writeColumns(b, "Flags", rows)
}

func (s *spec) writeGroup(b *strings.Builder, title string, fields []field) {
	rows := make([]usageRow, len(fields))
	for i, f := range fields {
		rows[i] = usageRow{label: usageLabel(f), help: s.lineHelp(f)}
	}
	writeColumns(b, title, rows)
}

type usageRow struct {
	label string
	help  string
}

func writeColumns(b *strings.Builder, title string, rows []usageRow) {
	if len(rows) == 0 {
		return
	}
	width := 0
	for _, row := range rows {
		if n := len(row.label); n > width {
			width = n
		}
	}
	fmt.Fprintf(b, "\n%s:\n", title)
	for _, row := range rows {
		fmt.Fprintf(b, "  %-*s  %s\n", width, row.label, row.help)
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
