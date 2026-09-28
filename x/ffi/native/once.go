package native

import "sync"

// Singleton runs load once on the calling goroutine and keeps the result.
// Later calls return that same value and error. Concurrent callers share the first call.
//
// The init stays on the caller. x/singleton starts it on another goroutine,
// which is the wrong thread for a library constructor.
func Singleton[T any](load func() (T, error)) func() (T, error) {
	return sync.OnceValues(load)
}

// Once is the error singleton. Later calls return the first error.
func Once(load func() error) func() error {
	run := Singleton(func() (struct{}, error) {
		return struct{}{}, load()
	})
	return func() error {
		_, err := run()
		return err
	}
}
