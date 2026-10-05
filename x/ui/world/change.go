package world

import "reflect"

// outsideReader is the cursor key for [Messages] called between systems.
const outsideReader = -1

func same[T any](a, b T) bool {
	var zero T
	if reflect.TypeOf(&zero).Elem().Comparable() {
		return any(a) == any(b)
	}
	return reflect.DeepEqual(a, b)
}

// mark records a real write at the current clock. A clock of zero is the
// baseline before the first frame, so that write is not a change.
func (w *World) mark(tick *uint64) {
	if w == nil || tick == nil || w.clock == 0 {
		return
	}
	*tick = w.clock
}

func (w *World) watch(fn func()) {
	if w == nil || fn == nil {
		return
	}
	w.watches = append(w.watches, fn)
}

func (w *World) evalWatches() {
	if w == nil {
		return
	}
	for _, fn := range w.watches {
		fn()
	}
}

// settle applies pending writes and drops them.
func (w *World) settle() {
	if w == nil {
		return
	}
	w.evalWatches()
	w.watches = nil
}

// changed reports a write this reader has not run since.
// Inside a system the reader is that system. Outside, it is the last frame.
func (w *World) changed(tick *uint64) bool {
	if w == nil || tick == nil {
		return false
	}
	w.evalWatches()
	if *tick == 0 {
		return false
	}
	if w.inSystem {
		return *tick > w.sysLast
	}
	return *tick > w.seen
}
