package world

type key[T any] struct{}

type row[T any] struct {
	gen   uint32
	stamp uint64
	tick  uint64
	v     T
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
// A value that differs from the stored one is marked written.
// The stamp moves on every insert, so a remove queued earlier in this
// system does not drop the value this insert just stored.
func (t *Table[T]) Insert(e Entity, v T) {
	if t == nil || t.w == nil || !t.w.Alive(e) {
		return
	}
	t.grow(e.Index)
	entry := t.sparse[e.Index]
	if entry == nil {
		entry = &row[T]{gen: e.Gen, stamp: 1, v: v}
		t.sparse[e.Index] = entry
		t.ids = append(t.ids, e.Index)
		t.w.mark(&entry.tick)
		return
	}
	sameGen := entry.gen == e.Gen
	old := entry.v
	entry.gen = e.Gen
	entry.v = v
	entry.stamp++
	if !sameGen || !same(old, v) {
		t.w.mark(&entry.tick)
	}
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

// Mut returns the stored value. The row is marked written when the
// caller changes it.
func (t *Table[T]) Mut(e Entity) *T {
	entry := t.lookup(e)
	if entry == nil {
		return nil
	}
	before := entry.v
	t.w.watch(func() {
		if !same(before, entry.v) {
			t.w.mark(&entry.tick)
		}
	})
	return &entry.v
}

// Changed reports that e was written since this reader last ran.
// Inside a system, the reader is that system. Outside, it is the last frame.
func (t *Table[T]) Changed(e Entity) bool {
	entry := t.lookup(e)
	if entry == nil {
		return false
	}
	return t.w.changed(&entry.tick)
}

// Each calls fn for each stored entity. A value fn changes is marked
// written. Entities inserted during fn are not visited. Despawn and
// Remove apply after the enclosing system.
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
		before := entry.v
		fn(e, &entry.v)
		if !same(before, entry.v) {
			t.w.mark(&entry.tick)
		}
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
// the system returns. An Insert of this column on e before that return
// keeps the inserted value.
func (t *Table[T]) Remove(e Entity) {
	if t == nil || t.w == nil {
		return
	}
	entry := t.lookup(e)
	if entry == nil {
		return
	}
	stamp := entry.stamp
	t.w.Defer(func() { t.clearStamp(e, stamp) })
}

// Join calls fn for each entity that has both A and B. A value fn
// changes is marked written. A and B are columns of one world.
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
		beforeA, beforeB := left.v, right.v
		fn(e, &left.v, &right.v)
		if !same(beforeA, left.v) {
			a.w.mark(&left.tick)
		}
		if !same(beforeB, right.v) {
			b.w.mark(&right.tick)
		}
	}
}

func (t *Table[T]) clear(e Entity) {
	t.drop(e, 0, false)
}

func (t *Table[T]) clearStamp(e Entity, stamp uint64) {
	t.drop(e, stamp, true)
}

func (t *Table[T]) drop(e Entity, stamp uint64, checkStamp bool) {
	if t == nil || int(e.Index) >= len(t.sparse) {
		return
	}
	entry := t.sparse[e.Index]
	if entry == nil || entry.gen != e.Gen {
		return
	}
	if checkStamp && entry.stamp != stamp {
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
