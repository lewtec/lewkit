package jni

import "sync/atomic"

var (
	javaRaw atomic.Uintptr
	runSlot atomic.Value
)

// setRunner installs the queue that runs work on the thread that called Bind.
func setRunner(fn func(func())) {
	if fn == nil {
		return
	}
	runSlot.Store(fn)
}

func currentRunner() func(func()) {
	fn, ok := runSlot.Load().(func(func()))
	if !ok {
		return nil
	}
	return fn
}

// runOnJava runs fn on the Bind thread when a runner is installed.
// Otherwise it runs fn on the caller.
func runOnJava(fn func()) {
	if fn == nil {
		return
	}
	if run := currentRunner(); run != nil {
		run(fn)
		return
	}
	fn()
}
