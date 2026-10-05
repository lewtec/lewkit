package world

// View is the rows of one column. [Query] returns it. [With] and
// [Without] narrow the rows. This is a Bevy query: the column is the
// stored Go type, and the filters are the other columns on that entity.
type View[T any] struct {
	w           *World
	hide        []func(Entity) bool
	onlyChanged bool
}

// Query returns the rows of T on w.
func Query[T any](w *World) View[T] {
	return View[T]{w: w}
}

// With keeps rows of T that also store U.
func With[T, U any](v View[T]) View[T] {
	if v.w == nil {
		return v
	}
	col := Column[U](v.w)
	next := v
	next.hide = append(append([]func(Entity) bool{}, v.hide...), func(e Entity) bool {
		_, ok := col.Get(e)
		return !ok
	})
	return next
}

// Without keeps rows of T that do not store U.
func Without[T, U any](v View[T]) View[T] {
	if v.w == nil {
		return v
	}
	col := Column[U](v.w)
	next := v
	next.hide = append(append([]func(Entity) bool{}, v.hide...), func(e Entity) bool {
		_, ok := col.Get(e)
		return ok
	})
	return next
}

// Changed keeps rows of T written since this reader last ran.
func Changed[T any](v View[T]) View[T] {
	v.onlyChanged = true
	return v
}

// Get copies the value stored for e when e passes the filters.
func (v View[T]) Get(e Entity) (T, bool) {
	var zero T
	entry := v.row(e)
	if entry == nil {
		return zero, false
	}
	return entry.v, true
}

// Mut returns the stored value when e passes the filters. The row is
// marked written when the caller changes it. Nil means the row is absent
// or filtered out.
func (v View[T]) Mut(e Entity) *T {
	if v.row(e) == nil {
		return nil
	}
	return Column[T](v.w).Mut(e)
}

// Read walks the filtered rows without marking them written. Rows
// inserted during fn are not visited.
func (v View[T]) Read(fn func(Entity, T)) {
	if fn == nil {
		return
	}
	v.each(false, func(e Entity, value *T) { fn(e, *value) })
}

// Each walks the filtered rows. A value fn changes is marked written.
// Rows inserted during fn are not visited.
func (v View[T]) Each(fn func(Entity, *T)) {
	v.each(true, fn)
}

// First returns the first filtered row that matches. A nil match accepts
// the first filtered row.
func (v View[T]) First(match func(T) bool) (Entity, T, bool) {
	var zero T
	if v.w == nil {
		return Entity{}, zero, false
	}
	table := Column[T](v.w)
	n := len(table.ids)
	for i := 0; i < n; i++ {
		index := table.ids[i]
		entry := table.sparse[index]
		if entry == nil {
			continue
		}
		e := Entity{Index: index, Gen: entry.gen}
		if v.row(e) == nil {
			continue
		}
		if match != nil && !match(entry.v) {
			continue
		}
		return e, entry.v, true
	}
	return Entity{}, zero, false
}

// All copies the filtered values.
func (v View[T]) All() []T {
	var out []T
	v.Read(func(_ Entity, value T) { out = append(out, value) })
	return out
}

func (v View[T]) row(e Entity) *row[T] {
	if v.w == nil || !v.w.Alive(e) {
		return nil
	}
	entry := Column[T](v.w).lookup(e)
	if entry == nil {
		return nil
	}
	for _, hide := range v.hide {
		if hide(e) {
			return nil
		}
	}
	if v.onlyChanged && !v.w.changed(&entry.tick) {
		return nil
	}
	return entry
}

func (v View[T]) each(edit bool, fn func(Entity, *T)) {
	if v.w == nil || fn == nil {
		return
	}
	table := Column[T](v.w)
	n := len(table.ids)
	for i := 0; i < n; i++ {
		index := table.ids[i]
		entry := table.sparse[index]
		if entry == nil {
			continue
		}
		e := Entity{Index: index, Gen: entry.gen}
		if v.row(e) == nil {
			continue
		}
		if !edit {
			fn(e, &entry.v)
			continue
		}
		before := entry.v
		fn(e, &entry.v)
		if !same(before, entry.v) {
			v.w.mark(&entry.tick)
		}
	}
}
