package cmd

import (
	"fmt"
	"reflect"
	"strings"
)

// Usage is the help text for T's flags, commands, and positionals.
func Usage[T any](name string) (string, error) {
	var zero T
	s, err := newSpec(reflect.ValueOf(&zero).Elem())
	if err != nil {
		return "", err
	}
	return s.usage(name, descriptionOf[T]()), nil
}

func descriptionOf[T any]() string {
	var zero T
	if d, ok := any(&zero).(Describer); ok {
		return d.Description()
	}
	if d, ok := any(zero).(Describer); ok {
		return d.Description()
	}
	return ""
}

func (s *spec) usage(name, desc string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage:\n  %s [flags]", name)
	if len(s.cmds) > 0 {
		fmt.Fprintf(&b, "\n  %s <command> [args]", name)
	}
	if len(s.pos) > 0 {
		fmt.Fprintf(&b, "\n  %s [flags]", name)
		for _, i := range s.pos {
			f := s.fields[i]
			if f.kind == kindRest {
				fmt.Fprintf(&b, " [%s...]", positionalName(f))
			} else {
				fmt.Fprintf(&b, " <%s>", positionalName(f))
			}
		}
	}
	b.WriteByte('\n')
	if desc != "" {
		b.WriteByte('\n')
		b.WriteString(desc)
		if !strings.HasSuffix(desc, "\n") {
			b.WriteByte('\n')
		}
	}

	var flags, cmds, pos []field
	for _, f := range s.fields {
		switch f.kind {
		case kindCommand:
			cmds = append(cmds, f)
		case kindPositional, kindRest:
			pos = append(pos, f)
		default:
			flags = append(flags, f)
		}
	}
	s.writeGroup(&b, "Flags", flags)
	s.writeGroup(&b, "Commands", cmds)
	s.writeGroup(&b, "Arguments", pos)
	return b.String()
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
	if !f.hasDef {
		return help
	}
	def := "(default: " + f.def + ")"
	if help == "" {
		return def
	}
	return help + " " + def
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
	case kindPositional, kindRest:
		return positionalName(f)
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
