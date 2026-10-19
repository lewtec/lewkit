//go:build ios

package std

import "github.com/lewtec/lewkit/x/ffi/native/dispatch"

// wake runs jobs posted from another goroutine while CFRunLoopRun is active.
// The scheme-task completion waits on this queue, and WebKit waits on that completion.
func (t *backend) wake() {
	dispatch.PostMain(func() {
		if !t.On() {
			return
		}
		t.drain()
	})
}
