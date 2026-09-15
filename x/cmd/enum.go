package cmd

import (
	"fmt"
	"strings"
)

// Enum is a string-backed closed set. Values lists the members.
type Enum[T ~string] interface {
	~string
	Values() []T
}

// EnumArg parses a CLI token into T. Unknown values are ErrInvalidArgument.
type EnumArg[T Enum[T]] struct {
	Container[T]
}

func (e *EnumArg[T]) Parse(arg string) error {
	var zero T
	want := zero.Values()
	for _, v := range want {
		if string(v) == arg {
			e.value = v
			return nil
		}
	}
	return fmt.Errorf("%w: want one of %s", ErrInvalidArgument, strings.Join(enumNames(want), ", "))
}

func (EnumArg[T]) ArgChoices() []string {
	var zero T
	return enumNames(zero.Values())
}

func enumNames[T ~string](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}
