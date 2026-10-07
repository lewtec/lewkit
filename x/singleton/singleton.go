package singleton

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Singleton[T any] interface {
	GetContext(ctx context.Context) (T, error)
}

func NewSingleton[T any](f func(context.Context) (T, error)) Singleton[T] {
	var first atomic.Pointer[context.Context]
	init := sync.OnceValues(func() (T, error) {
		return f(*first.Load())
	})
	return singleton[T](func(ctx context.Context) (T, error) {
		if ctx == nil {
			var z T
			return z, errors.New("singleton: nil context")
		}
		if err := ctx.Err(); err != nil {
			var z T
			return z, context.Cause(ctx)
		}
		first.CompareAndSwap(nil, &ctx)
		done := make(chan struct {
			v   T
			err error
		}, 1)
		go func() {
			v, err := init()
			done <- struct {
				v   T
				err error
			}{v, err}
		}()
		select {
		case <-ctx.Done():
			var z T
			return z, context.Cause(ctx)
		case r := <-done:
			return r.v, r.err
		}
	})
}

func NewSingletonFunc[T any](f func(context.Context) (T, error)) func(context.Context) T {
	s := NewSingleton(f)
	return func(ctx context.Context) T {
		return MustGet(s, ctx)
	}
}

type singleton[T any] func(context.Context) (T, error)

func (s singleton[T]) GetContext(ctx context.Context) (T, error) {
	return s(ctx)
}

func MustGet[T any](s Singleton[T], ctx context.Context) T {
	ret, err := s.GetContext(ctx)
	if err != nil {
		panic(err)
	}
	return ret
}
