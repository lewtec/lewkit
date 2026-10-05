package world

// Entity is one generational index. The zero Entity is never alive.
type Entity struct {
	Index uint32
	Gen   uint32
}

type slot struct {
	gen   uint32
	alive bool
}

type column interface {
	clear(Entity)
}

type roller interface {
	roll(frame uint64)
}

// World is the entities, columns, resources, and messages of one [Sim].
// Use it from the goroutine that calls [Sim.Frame].
type World struct {
	frame    uint64
	clock    uint64
	seen     uint64
	sysLast  uint64
	reader   int
	inFrame  bool
	inSystem bool
	flushing bool
	queue    []func()
	watches  []func()
	slots    []slot
	free     []uint32
	cols     map[any]column
	clean    []column
	res      map[any]any
	msgs     map[any]roller
	rolls    []roller
}

func newWorld() *World {
	return &World{
		reader: outsideReader,
		cols:   map[any]column{},
		res:    map[any]any{},
		msgs:   map[any]roller{},
	}
}

// Tick is the current frame. It is zero before the first [Sim.Frame].
func (w *World) Tick() uint64 {
	if w == nil {
		return 0
	}
	return w.frame
}

// Spawn returns a live entity. During a system the id is live immediately.
func (w *World) Spawn() Entity {
	if w == nil {
		return Entity{}
	}
	if n := len(w.free); n > 0 {
		index := w.free[n-1]
		w.free = w.free[:n-1]
		w.slots[index].alive = true
		return Entity{Index: index, Gen: w.slots[index].gen}
	}
	w.slots = append(w.slots, slot{gen: 1, alive: true})
	return Entity{Index: uint32(len(w.slots) - 1), Gen: 1}
}

// Alive reports whether e was spawned and has not been despawned.
func (w *World) Alive(e Entity) bool {
	if w == nil || e.Gen == 0 || int(e.Index) >= len(w.slots) {
		return false
	}
	s := w.slots[e.Index]
	return s.alive && s.gen == e.Gen
}

// Despawn recycles e. During a system the entity stays alive until the
// system returns, then its columns drop that row.
func (w *World) Despawn(e Entity) {
	if w == nil {
		return
	}
	w.Defer(func() { w.despawnNow(e) })
}

// Defer runs fn after the current system, in queue order.
// A Defer made while the queue is flushing waits for the next batch.
// Outside a system, and outside a flush, fn runs now.
func (w *World) Defer(fn func()) {
	if w == nil || fn == nil {
		return
	}
	if w.inSystem || w.flushing {
		w.queue = append(w.queue, fn)
		return
	}
	fn()
}

func (w *World) despawnNow(e Entity) {
	if !w.Alive(e) {
		return
	}
	for _, col := range w.clean {
		col.clear(e)
	}
	w.slots[e.Index].alive = false
	gen := w.slots[e.Index].gen + 1
	if gen == 0 {
		gen = 1
	}
	w.slots[e.Index].gen = gen
	w.free = append(w.free, e.Index)
}

func (w *World) flush() {
	if w == nil || w.flushing {
		return
	}
	w.flushing = true
	for len(w.queue) > 0 {
		batch := w.queue
		w.queue = nil
		for _, fn := range batch {
			fn()
		}
	}
	w.flushing = false
	w.settle()
}

func (w *World) roll() {
	for _, box := range w.rolls {
		box.roll(w.frame)
	}
}
