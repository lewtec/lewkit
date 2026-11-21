package future

// Future represents a calculation that is not necessarily available now
type Future[T any] interface {
	// State gets wether the future is running or which state it is
	State() futureState
	// Tick is used internally to wake up a future for execution
	Tick() (done bool)
	// Peek gets the internal values, ErrNotResolved if the computation isn't done yet
	Peek() (T, error)
	// Get executes the future
	Get() (T, error)
	// Resolve is like Get but only returns the error
	Resolve() error
}
