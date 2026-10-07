package thread

import (
	"context"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	mu sync.Mutex
	ui Driver
)

// open selects the UI-thread driver from ctx. Later calls return that driver.
func open(ctx context.Context) Driver {
	if ctx == nil {
		panic("thread: nil context")
	}
	mu.Lock()
	d := ui
	mu.Unlock()
	if d != nil {
		return d
	}
	created, err := driver.Get[Driver](ctx)
	if err != nil {
		panic(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if ui == nil {
		ui = created
	}
	return ui
}

func current() Driver {
	mu.Lock()
	defer mu.Unlock()
	if ui == nil {
		panic("thread: driver not open")
	}
	return ui
}

// Bind locks this goroutine to the UI thread. Call from main.
// ctx selects the driver. A nil context panics.
func Bind(ctx context.Context) { open(ctx).Bind() }

// Bound reports whether Bind has been called.
func Bound() bool { return current().Bound() }

// On reports whether this goroutine is on the UI thread.
func On() bool { return current().On() }

// OnIdle registers fn on the UI thread when Loop has no job.
func OnIdle(fn func()) { current().OnIdle(fn) }

// Loop serves the UI thread until ctx is done.
func Loop(ctx context.Context) { open(ctx).Loop(ctx) }

// Do runs fn on the UI thread and waits.
func Do(fn func()) { current().Do(fn) }

// Go queues fn on the UI thread and returns.
func Go(fn func()) { current().Go(fn) }

// Enqueue queues fn on the UI thread without blocking the caller.
func Enqueue(fn func()) { current().Enqueue(fn) }
