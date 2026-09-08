package singleton

import (
	"context"
	"sync"
)

type Singleton[T any] interface {
	GetContext(ctx context.Context) (T, error)
}

func NewSingleton[T any](f func(context.Context) (T, error)) Singleton[T] {
	return &naiveSingleton[T]{}
}

func NewSingletonFunc[T any](f func(context.Context) (T, error)) func() T {
	singleton := NewSingleton(f)
	return func() T {
		MustGet(singleton, context.Background())
	}
}

type naiveSingleton[T any] struct {
	item    T
	err     error
	handler func(context.Context) (T, error)
	once    sync.Once
}

func (s *naiveSingleton[T]) GetContext(ctx context.Context) (T, error) {
	s.once.Do(func() {
		s.item, s.err = s.handler(ctx)
	})
	return s.item, s.err
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
