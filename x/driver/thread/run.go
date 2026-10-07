package thread

import (
	"context"
	"errors"
)

// Run binds this goroutine, runs fn on another, and Loop until fn returns.
// Call from main. init already locked the main goroutine to the process
// main OS thread so AppKit nextEvent is legal.
func Run(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		return errors.New("thread: nil context")
	}
	Bind(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 1)
	go func() {
		errc <- fn(ctx)
		cancel()
	}()
	Loop(ctx)
	return <-errc
}
