package sops

import (
	"fmt"
	"strings"
)

// Kind names a tool target.
type Kind string

// KindConda runs the command with conda run -n.
const KindConda Kind = "conda"

// Target is a tool and the name to run inside it.
// A zero Target runs the command directly.
type Target struct {
	Kind Kind
	Name string
}

// ParseTarget reads kind:name. The name keeps any colons after the first.
// An empty string is a zero Target.
func ParseTarget(s string) (Target, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Target{}, nil
	}
	kind, name, ok := strings.Cut(s, ":")
	if !ok || kind == "" || name == "" {
		return Target{}, fmt.Errorf("sops: target %q must be kind:name", s)
	}
	switch Kind(kind) {
	case KindConda:
		return Target{Kind: KindConda, Name: name}, nil
	default:
		return Target{}, fmt.Errorf("sops: unknown target kind %q", kind)
	}
}

// Command returns the program and arguments that run args inside the target.
func (t Target) Command(args []string) (string, []string, error) {
	switch t.Kind {
	case "":
		if len(args) == 0 {
			return "", nil, fmt.Errorf("sops: command is empty")
		}
		return args[0], append([]string(nil), args[1:]...), nil
	case KindConda:
		if t.Name == "" {
			return "", nil, fmt.Errorf("sops: conda target needs a name")
		}
		argv := make([]string, 0, 5+len(args))
		argv = append(argv, "run", "-n", t.Name, "--no-capture-output", "--")
		argv = append(argv, args...)
		return "conda", argv, nil
	default:
		return "", nil, fmt.Errorf("sops: unknown target kind %q", t.Kind)
	}
}
