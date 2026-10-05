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

func open(ctx context.Context) Driver {
	once.Do(func() {
		if ctx == nil {
			panic("thread: nil context")
		}
		d, err := driver.Get[Driver](ctx)
		if err != nil {
			panic(err)
		}
		ui = d
	})
	return ui
}

func current() Driver {
	if ui == nil {
		panic("thread: used before Run or Bind")
	}
	return ui
}

// Bind locks this goroutine to the UI thread.
// Run resolves the driver from the process context. Bind is the root
// for a host that has no parent context, such as the Android loader.
func Bind() { open(context.Background()).Bind() }

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
