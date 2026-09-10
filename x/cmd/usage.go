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
	return s.usage(name), nil
}

func (s *spec) usage(name string) string {
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
	writeGroup(&b, "Flags", flags)
	writeGroup(&b, "Commands", cmds)
	writeGroup(&b, "Arguments", pos)
	return b.String()
}

func writeGroup(b *strings.Builder, title string, fields []field) {
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
		fmt.Fprintf(b, "  %-*s  %s\n", width, labels[i], f.help)
	}
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
