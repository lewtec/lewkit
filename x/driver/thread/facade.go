package thread

import (
	"context"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	once sync.Once
	ui   Driver
)

func current() Driver {
	once.Do(func() {
		d, err := driver.Get[Driver](context.Background())
		if err != nil {
			panic(err)
		}
		ui = d
	})
	return ui
}

// Bind locks this goroutine to the UI thread. Call from main.
func Bind() { current().Bind() }

// Bound reports whether Bind has been called.
func Bound() bool { return current().Bound() }

// On reports whether this goroutine is on the UI thread.
func On() bool { return current().On() }

// OnIdle registers fn on the UI thread when Loop has no job.
func OnIdle(fn func()) { current().OnIdle(fn) }

// Loop serves the UI thread until ctx is done.
func Loop(ctx context.Context) { current().Loop(ctx) }

// Do runs fn on the UI thread and waits.
func Do(fn func()) { current().Do(fn) }

// Go queues fn on the UI thread and returns.
func Go(fn func()) { current().Go(fn) }

// Enqueue queues fn on the UI thread without blocking the caller.
func Enqueue(fn func()) { current().Enqueue(fn) }
