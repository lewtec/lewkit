//go:build darwin && !ios

package dispatch

// OnMain runs fn on the caller. macOS UI work uses the process main thread.
func OnMain(fn func()) {
	if fn != nil {
		fn()
	}
}
