package future

type futureState uint32

const (
	FuturePending futureState = iota
	FutureRunning
	FutureIdle
	FutureCancelled
	FutureError
	FutureSuccess
)

func (f futureState) IsPending() bool {
	return f == FuturePending
}

func (f futureState) IsResolved() bool {
	return f == FutureError || f == FutureSuccess
}

func (f futureState) IsRunning() bool {
	return f == FutureRunning
}

func (f futureState) IsCancelled() bool {
	return f == FutureCancelled
}

func (f futureState) asNative() uint32 {
	return uint32(f)
}
