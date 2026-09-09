package cmd

import "context"

type Action[T any] interface {
	Run(ctx context.Context, args Command[T]) error
}

type Command[T any] struct {
	args T
}

func (c *Command[T]) Parse(args ...string) error {
	return nil
}
