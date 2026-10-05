package world

type messages[T any] struct {
	items   []T
	born    []uint64
	start   uint64
	next    uint64
	cursors map[int]uint64
}

func (m *messages[T]) roll(frame uint64) {
	if frame < 2 {
		return
	}
	keep := frame - 1
	drop := 0
	for drop < len(m.born) && m.born[drop] < keep {
		drop++
	}
	if drop == 0 {
		return
	}
	m.items = m.items[drop:]
	m.born = m.born[drop:]
	m.start += uint64(drop)
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

// Send appends v to the log of T.
// Inside a frame, v belongs to that frame. Outside, it belongs to the next one.
// The frame after next drops it. A system that has not read it yet still
// observes it on its next run.
func Send[T any](w *World, v T) {
	if w == nil {
		return
	}
	box := inbox[T](w)
	born := w.frame
	if !w.inFrame {
		born = w.frame + 1
	}
	box.items = append(box.items, v)
	box.born = append(box.born, born)
	box.next = box.start + uint64(len(box.items))
}

// Messages returns a copy of the values of T this reader has not seen,
// then marks them seen. A second call in the same system is empty until
// a later [Send]. The copy is the caller's.
func Messages[T any](w *World) []T {
	if w == nil {
		return nil
	}
	box, ok := w.msgs[key[T]{}].(*messages[T])
	if !ok {
		return nil
	}
	if box.cursors == nil {
		box.cursors = map[int]uint64{}
	}
	cur := box.cursors[w.reader]
	var out []T
	for i := range box.items {
		id := box.start + uint64(i)
		if id < cur {
			continue
		}
		out = append(out, box.items[i])
	}
	box.cursors[w.reader] = box.next
	return out
}
