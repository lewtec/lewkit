package world

type key[T any] struct{}

type row[T any] struct {
	gen  uint32
	tick uint64
	v    T
}

// Table is the column of T. [Column] returns the one table for that type.
type Table[T any] struct {
	w      *World
	sparse []*row[T]
	ids    []uint32
}

// Column returns the table of T on w, creating it on first use.
func Column[T any](w *World) *Table[T] {
	if w == nil {
		return nil
	}
	k := key[T]{}
	if table, ok := w.cols[k].(*Table[T]); ok {
		return table
	}
	table := &Table[T]{w: w}
	w.cols[k] = table
	w.clean = append(w.clean, table)
	return table
}

// Len is the number of stored values, including rows a deferred despawn
// has not cleared yet.
func (t *Table[T]) Len() int {
	if t == nil {
		return 0
	}
	return len(t.ids)
}

// Insert stores v on a live entity. A dead entity is left unchanged.
// The value is marked written this frame.
func (t *Table[T]) Insert(e Entity, v T) {
	if t == nil || t.w == nil || !t.w.Alive(e) {
		return
	}
	t.grow(e.Index)
	entry := t.sparse[e.Index]
	if entry == nil {
		entry = &row[T]{}
		t.sparse[e.Index] = entry
		t.ids = append(t.ids, e.Index)
	}
	entry.gen = e.Gen
	entry.v = v
	entry.tick = t.w.tick
}

// Get copies the value stored for e.
func (t *Table[T]) Get(e Entity) (T, bool) {
	var zero T
	entry := t.lookup(e)
	if entry == nil {
		return zero, false
	}
	return entry.v, true
}

// Mut returns the stored value and marks it written this frame.
func (t *Table[T]) Mut(e Entity) *T {
	entry := t.lookup(e)
	if entry == nil {
		return nil
	}
	entry.tick = t.w.tick
	return &entry.v
}

// Changed reports that Insert, Mut, or Each wrote e during this frame.
func (t *Table[T]) Changed(e Entity) bool {
	entry := t.lookup(e)
	if entry == nil || t.w.tick == 0 {
		return false
	}
	return entry.tick == t.w.tick
}

// Each calls fn for each stored entity. fn may change the value, and
// each visit marks that value written. Entities inserted during fn are
// not visited. Despawn and Remove apply after the enclosing system.
func (t *Table[T]) Each(fn func(Entity, *T)) {
	if t == nil || t.w == nil || fn == nil {
		return
	}
	n := len(t.ids)
	for i := 0; i < n; i++ {
		index := t.ids[i]
		entry := t.sparse[index]
		if entry == nil {
			continue
		}
		e := Entity{Index: index, Gen: entry.gen}
		if !t.w.Alive(e) {
			continue
		}
		fn(e, &entry.v)
		entry.tick = t.w.tick
	}
}

// Read walks stored values without marking them written.
func (t *Table[T]) Read(fn func(Entity, T)) {
	if t == nil || t.w == nil || fn == nil {
		return
	}
	n := len(t.ids)
	for i := 0; i < n; i++ {
		index := t.ids[i]
		entry := t.sparse[index]
		if entry == nil {
			continue
		}
		e := Entity{Index: index, Gen: entry.gen}
		if !t.w.Alive(e) {
			continue
		}
		fn(e, entry.v)
	}
}

// Remove drops e from this column. During a system the row stays until
// the system returns.
func (t *Table[T]) Remove(e Entity) {
	if t == nil || t.w == nil {
		return
	}
	t.w.Defer(func() { t.clear(e) })
}

// Join calls fn for each entity that has both A and B. Both values are
// mutable and both are marked written. A and B are columns of one world.
// Entities inserted during fn are not visited.
func Join[A, B any](a *Table[A], b *Table[B], fn func(Entity, *A, *B)) {
	if a == nil || b == nil || a.w == nil || a.w != b.w || fn == nil {
		return
	}
	n := len(a.ids)
	for i := 0; i < n; i++ {
		index := a.ids[i]
		left := a.sparse[index]
		if left == nil || int(index) >= len(b.sparse) {
			continue
		}
		right := b.sparse[index]
		if right == nil || right.gen != left.gen {
			continue
		}
		e := Entity{Index: index, Gen: left.gen}
		if !a.w.Alive(e) {
			continue
		}
		fn(e, &left.v, &right.v)
		left.tick = a.w.tick
		right.tick = b.w.tick
	}
}

func (t *Table[T]) clear(e Entity) {
	if t == nil || int(e.Index) >= len(t.sparse) {
		return
	}
	entry := t.sparse[e.Index]
	if entry == nil || entry.gen != e.Gen {
		return
	}
	t.sparse[e.Index] = nil
	for i, index := range t.ids {
		if index != e.Index {
			continue
		}
		last := len(t.ids) - 1
		t.ids[i] = t.ids[last]
		t.ids = t.ids[:last]
		return
	}
}

func (t *Table[T]) lookup(e Entity) *row[T] {
	if t == nil || t.w == nil || !t.w.Alive(e) || int(e.Index) >= len(t.sparse) {
		return nil
	}
	entry := t.sparse[e.Index]
	if entry == nil || entry.gen != e.Gen {
		return nil
	}
	return entry
}

func (t *Table[T]) grow(index uint32) {
	if int(index) < len(t.sparse) {
		return
	}
	next := make([]*row[T], index+1)
	copy(next, t.sparse)
	t.sparse = next
}
