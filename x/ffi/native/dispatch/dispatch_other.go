//go:build !darwin

package dispatch

// OnMain runs fn on the caller. This build has no Apple main queue.
func OnMain(fn func()) {
	if fn != nil {
		fn()
	}
}
