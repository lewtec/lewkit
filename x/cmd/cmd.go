package cmd

import (
	"context"
	"reflect"
)

type Action[T any] interface {
	Run(ctx context.Context, args Command[T]) error
}

// Describer is a command spec that supplies a paragraph for Usage.
type Describer interface {
	Description() string
}

type Command[T any] struct {
	args T
}

func Parse[T any](args ...string) (T, error) {
	var c Command[T]
	err := c.Parse(args...)
	return c.Args(), err
}

func (c *Command[T]) Args() T {
	return c.args
}

func (c *Command[T]) Parse(args ...string) error {
	var zero T
	c.args = zero
	return parseArgs(reflect.ValueOf(&c.args).Elem(), args)
}
