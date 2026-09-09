package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/exp/constraints"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnknownFlag     = errors.New("unknown flag")
	ErrMissingValue    = errors.New("missing value")
	ErrInvalidSpec     = errors.New("invalid command spec")
)

type Parser interface {
	Parse(string) error
}

type Counter interface {
	Count(int) error
}

type Valuer[T any] interface {
	Value() T
}

func Values[T any, A Valuer[T]](in []A) []T {
	out := make([]T, len(in))
	for i, a := range in {
		out[i] = a.Value()
	}
	return out
}

type Container[T any] struct {
	value T
}

func (c Container[T]) Value() T {
	return c.value
}

type StringArg struct {
	Container[string]
}

func (s *StringArg) Parse(arg string) error {
	s.value = arg
	return nil
}

type IntArg[T constraints.Integer] struct {
	Container[T]
}

func (i *IntArg[T]) Parse(arg string) error {
	value, err := strconv.ParseInt(arg, 0, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	i.value = T(value)
	return err
}

type FloatArg[T constraints.Float] struct {
	Container[T]
}

func (i *FloatArg[T]) Parse(arg string) error {
	value, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	i.value = T(value)
	return err
}

type Flag struct {
	Container[bool]
}

func (f *Flag) Count(times int) error {
	f.value = times != 0
	if times > 1 {
		return fmt.Errorf("%w: flags must only be used once", ErrInvalidArgument)
	}
	return nil
}

type Count struct {
	Container[int]
}

func (f *Count) Count(times int) error {
	f.value = times
	return nil
}

func (f *Count) Parse(arg string) error {
	value, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	if value < 0 {
		return fmt.Errorf("%w: count must be non-negative", ErrInvalidArgument)
	}
	f.value = value
	return nil
}

var (
	_ Parser       = (*StringArg)(nil)
	_ Parser       = (*IntArg[int])(nil)
	_ Parser       = (*FloatArg[float64])(nil)
	_ Parser       = (*Count)(nil)
	_ Counter      = (*Flag)(nil)
	_ Counter      = (*Count)(nil)
	_ Valuer[bool] = (*Flag)(nil)
)
