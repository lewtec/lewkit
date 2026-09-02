package future

type Future[T any] interface {
	State() futureState
	Peek() (T, error)
	Get() (T, error)
	Resolve() error
}
