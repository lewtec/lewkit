package world

type cell[T any] struct {
	v    T
	tick uint64
}

// Put stores the single value of T on w. The same cell stays in place,
// so a pointer from [Mut] still refers to it. An equal value is not a write.
func Put[T any](w *World, v T) {
	if w == nil {
		return
	}
	if c := resource[T](w); c != nil {
		if same(c.v, v) {
			return
		}
		c.v = v
		w.mark(&c.tick)
		return
	}
	c := &cell[T]{v: v}
	w.res[key[T]{}] = c
	w.mark(&c.tick)
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

// Mut returns the resource of T. The resource is marked written when the
// caller changes it. Nil means Put was not called for T.
func Mut[T any](w *World) *T {
	c := resource[T](w)
	if c == nil {
		return nil
	}
	before := c.v
	w.watch(func() {
		if !same(before, c.v) {
			w.mark(&c.tick)
		}
	})
	return &c.v
}

// Written reports that T was written since this reader last ran.
// Inside a system, the reader is that system. Outside, it is the last frame.
func Written[T any](w *World) bool {
	c := resource[T](w)
	if c == nil {
		return false
	}
	return w.changed(&c.tick)
}

func resource[T any](w *World) *cell[T] {
	if w == nil {
		return nil
	}
	c, _ := w.res[key[T]{}].(*cell[T])
	return c
}
