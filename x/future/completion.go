package future

import (
	"context"
	"errors"
	"sync/atomic"
)

func NewFuture[T any](ctx context.Context, handler func(ctx context.Context) (T, error)) Future[T] {
	ret := &completionFuture[T]{
		ctx:     ctx,
		handler: handler,
	}
	ret.state.Swap(FuturePending.asNative())
	return ret
}

type completionFuture[T any] struct {
	state   atomic.Uint32
	ctx     context.Context
	value   T
	err     error
	handler func(ctx context.Context) (T, error)
}

func (f *completionFuture[T]) State() futureState {
	return futureState(f.state.Load())
}

func (f *completionFuture[T]) Tick() bool {
	f.resolve()
	return true
}

func (f *completionFuture[T]) Peek() (v T, err error) {
	if !f.State().IsResolved() {
		err = ErrNotResolved
		return
	}
	v = f.value
	err = f.err
	return
}

func (f *completionFuture[T]) Resolve() error {
	f.resolve()
	return f.err
}

func (f *completionFuture[T]) Get() (T, error) {
	f.resolve()
	return f.value, f.err
}

func (f *completionFuture[T]) resolve() {
	if !f.state.CompareAndSwap(FuturePending.asNative(), FutureRunning.asNative()) {
		return
	}
	f.value, f.err = f.handler(f.ctx)
	if errors.Is(f.err, context.Canceled) {
		f.state.Swap(FutureCancelled.asNative())
		return
	}
	if f.err != nil {
		f.state.Swap(FutureError.asNative())
	} else {
		f.state.Swap(FutureSuccess.asNative())
	}
}
