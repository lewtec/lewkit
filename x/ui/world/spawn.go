package world

// Spawn stores v on a new entity and returns that entity. The row is
// visible to a later system in this frame. A dead world returns the zero
// [Entity].
func Spawn[T any](w *World, v T) Entity {
	if w == nil {
		return Entity{}
	}
	e := w.Spawn()
	Column[T](w).Insert(e, v)
	return e
}

// Spawn2 stores a and b on one new entity. This is a Bevy spawn of two
// columns. Further columns use [Insert].
func Spawn2[A, B any](w *World, a A, b B) Entity {
	e := Spawn(w, a)
	Insert(w, e, b)
	return e
}

// Insert stores v on a live entity. A dead entity is left unchanged.
func Insert[T any](w *World, e Entity, v T) {
	Column[T](w).Insert(e, v)
}

// Init stores v when T is not already on w. A second call leaves the
// first value in place, so two plugins can both declare the same resource.
func Init[T any](w *World, v T) {
	if w == nil {
		return
	}
	if _, ok := Read[T](w); ok {
		return
	}
	Put(w, v)
}
