package native

import "sync"

// Once returns a function that runs load on the first call.
// Later calls return that same error. Concurrent callers share the first call.
func Once(load func() error) func() error {
	run := sync.OnceValues(func() (struct{}, error) {
		return struct{}{}, load()
	})
	return func() error {
		_, err := run()
		return err
	}
}
