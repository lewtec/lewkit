package cmd

import (
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"
)

// Enum is an iota-style closed set. String is the CLI token; Values lists members.
type Enum[T any] interface {
	constraints.Integer
	fmt.Stringer
	Values() []T
}

// EnumArg parses a CLI token into T by matching String(). Unknown values are ErrInvalidArgument.
type EnumArg[T Enum[T]] struct {
	Container[T]
}

func (e *EnumArg[T]) Parse(arg string) error {
	var zero T
	want := zero.Values()
	for _, v := range want {
		if v.String() == arg {
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

func enumNames[T Enum[T]](vs []T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out
}
