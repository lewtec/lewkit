package singleton

import (
	"context"
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

func NewSingletonFunc[T any](f func(context.Context) (T, error)) func() T {
	return sync.OnceValue(func() T {
		v, err := f(context.Background())
		if err != nil {
			panic(err)
		}
		return v
	})
}

type singleton[T any] func(context.Context) (T, error)

func (s singleton[T]) GetContext(ctx context.Context) (T, error) {
	return s(ctx)
}

func Get[T any](s Singleton[T]) (T, error) {
	return s.GetContext(context.Background())
}

func MustGet[T any](s Singleton[T], ctx context.Context) T {
	ret, err := s.GetContext(ctx)
	if err != nil {
		panic(err)
	}
	return ret
}
