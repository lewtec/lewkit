package cmd

import (
	"fmt"
	"reflect"
	"strings"
)

// Ask is one flag the command still needs.
// Prompt, Default, and Choices come from the same spec as the CLI:
// help text, the default tag, and ArgChoices.
type Ask struct {
	Name    string
	Prompt  string
	Default string
	Choices []string
	apply   func(string) error
}

// Apply writes value through the flag's parser.
func (a Ask) Apply(value string) error {
	if a.apply == nil {
		return nil
	}
	return a.apply(value)
}

// Asks lists value flags that were not explicitly set, and explicitly set
// text flags whose value is still empty. Enum choices are ArgChoices.
// root is a pointer to the parsed command struct.
func Asks(root any) ([]Ask, error) {
	if root == nil {
		return nil, fmt.Errorf("%w: nil args", ErrInvalidSpec)
	}
	s, err := newSpec(reflect.ValueOf(root))
	if err != nil {
		return nil, err
	}
	var out []Ask
	for i, f := range s.fields {
		if !askKind(f.kind) {
			continue
		}
		fv := s.root.FieldByIndex(f.index)
		choices := append([]string(nil), choicesOf(fv.Type())...)
		if argSet(fv) && !blankText(argString(fv), choices) {
			continue
		}
		prompt := f.help
		if prompt == "" {
			prompt = f.display()
		}
		fi := i
		out = append(out, Ask{
			Name:    f.long,
			Prompt:  prompt,
			Default: f.def,
			Choices: choices,
			apply: func(val string) error {
				if err := s.setValue(s.fields[fi], val); err != nil {
					return err
				}
				noteSet(s.root.FieldByIndex(s.fields[fi].index))
				return nil
			},
		})
	}
	return out, nil
}

func askKind(k fieldKind) bool {
	switch k {
	case kindValue, kindEither, kindPositional:
		return true
	default:
		return false
	}
}

func blankText(value string, choices []string) bool {
	return len(choices) == 0 && strings.TrimSpace(value) == ""
}

func argSet(v reflect.Value) bool {
	m := rvalue{v}.ptr().MethodByName("ArgSet")
	if !m.IsValid() {
		return false
	}
	t := m.Type()
	if t.NumIn() != 0 || t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
		return false
	}
	return m.Call(nil)[0].Bool()
}

func argString(v reflect.Value) string {
	m := rvalue{v}.ptr().MethodByName("Value")
	if !m.IsValid() || m.Type().NumIn() != 0 || m.Type().NumOut() != 1 {
		return ""
	}
	val := m.Call(nil)[0]
	if val.Kind() == reflect.String {
		return val.String()
	}
	if val.CanInterface() {
		if s, ok := val.Interface().(fmt.Stringer); ok {
			return s.String()
		}
	}
	return ""
}
