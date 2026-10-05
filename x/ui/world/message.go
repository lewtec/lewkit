package world

type messages[T any] struct {
	items []T
}

func (m *messages[T]) roll() {
	m.items = m.items[:0]
}

func inbox[T any](w *World) *messages[T] {
	k := key[T]{}
	if box, ok := w.msgs[k].(*messages[T]); ok {
		return box
	}
	box := &messages[T]{}
	w.msgs[k] = box
	w.rolls = append(w.rolls, box)
	return box
}

// Send appends v to this frame's log of T. The next [Sim.Frame] drops it.
// A system in a later [Set] observes v. A system in an earlier set does not.
func Send[T any](w *World, v T) {
	if w == nil {
		return
	}
	box := inbox[T](w)
	box.items = append(box.items, v)
}

// Messages returns this frame's log of T.
// The slice is reused on the next [Sim.Frame].
func Messages[T any](w *World) []T {
	if w == nil {
		return nil
	}
	box, ok := w.msgs[key[T]{}].(*messages[T])
	if !ok {
		return nil
	}
	return box.items
}
