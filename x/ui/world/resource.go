package world

type cell[T any] struct {
	v    T
	tick uint64
}

// Put stores the single value of T on w and marks it written this frame.
func Put[T any](w *World, v T) {
	if w == nil {
		return
	}
	w.res[key[T]{}] = &cell[T]{v: v, tick: w.tick}
}

// Read copies the resource of T.
func Read[T any](w *World) (T, bool) {
	var zero T
	c := resource[T](w)
	if c == nil {
		return zero, false
	}
	return c.v, true
}

// Mut returns the resource of T and marks it written. Nil means Put was
// not called for T.
func Mut[T any](w *World) *T {
	c := resource[T](w)
	if c == nil {
		return nil
	}
	c.tick = w.tick
	return &c.v
}

// Written reports that Put or Mut touched T during this frame.
func Written[T any](w *World) bool {
	c := resource[T](w)
	return c != nil && w.tick != 0 && c.tick == w.tick
}

func resource[T any](w *World) *cell[T] {
	if w == nil {
		return nil
	}
	c, _ := w.res[key[T]{}].(*cell[T])
	return c
}
