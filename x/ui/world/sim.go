package world

import "context"

// Set is one pass of a frame. The constants run in declaration order.
// Startup runs on the first [Sim.Frame] only.
type Set uint8

const (
	Startup Set = iota
	First
	PreUpdate
	Update
	PostUpdate
	Last
	setCount
)

// System advances a [World]. It runs on the caller goroutine.
type System func(context.Context, *World)

// Plugin registers systems on a [Sim]. It is not a driver and it does
// not open a window.
type Plugin func(*Sim)

// Sim is the schedule for one [World]. Register plugins before the
// first [Sim.Frame].
type Sim struct {
	World   *World
	systems [setCount][]placed
	saved   [setCount][]int
	fresh   bool
	seq     int
	started bool
	delta   float64
	stepped bool
}

// New returns an empty simulation.
func New() *Sim {
	return &Sim{World: newWorld()}
}

// Plugin calls build so it can register systems.
func (s *Sim) Plugin(build Plugin) {
	if s == nil || build == nil {
		return
	}
	build(s)
}

// Group returns one plugin that registers each plugin in order.
// Edges between systems decide the frame, not the order of these plugins.
func Group(plugins ...Plugin) Plugin {
	return func(s *Sim) {
		for _, build := range plugins {
			s.Plugin(build)
		}
	}
}

// System appends sys to set. The returned [Reg] can name systems that
// must run before or after sys. A bad set panics. A nil sys returns nil.
func (s *Sim) System(set Set, sys System) *Reg {
	if s == nil || sys == nil {
		return nil
	}
	if set >= setCount {
		panic("world: bad set")
	}
	name := funcName(sys)
	if name == "" {
		panic("world: system has no name")
	}
	s.fresh = false
	for _, existing := range s.systems[set] {
		if existing.name == name {
			panic("world: duplicate system " + name)
		}
	}
	s.systems[set] = append(s.systems[set], placed{
		run:  sys,
		name: name,
		seq:  s.seq,
	})
	s.seq++
	return &Reg{sim: s, set: set, idx: len(s.systems[set]) - 1}
}

// Chain runs systems in the order written. A system added elsewhere can
// sit between them with [Reg.After] and [Reg.Before].
func (s *Sim) Chain(set Set, systems ...System) {
	if s == nil {
		return
	}
	var prev System
	for _, sys := range systems {
		if sys == nil {
			continue
		}
		reg := s.System(set, sys)
		if prev != nil {
			reg.After(prev)
		}
		prev = sys
	}
}

// Frame runs Startup once, then First through Last.
// Every set is ordered before the world changes. A nil ctx panics.
// A canceled ctx runs no further systems. The system that observed the
// cancel still flushes its deferred edits. A message is dropped on the
// frame after the one that follows its send.
func (s *Sim) Frame(ctx context.Context) {
	if s == nil || s.World == nil {
		if s != nil {
			s.stepped = false
		}
		return
	}
	if ctx == nil {
		panic("world: nil context")
	}
	plan := s.prepare()
	if ctx.Err() != nil {
		s.stepped = false
		return
	}
	w := s.World
	w.settle()
	w.frame++
	w.inFrame = true
	defer func() { w.inFrame = false }()
	w.clock++
	boundary := w.clock
	defer func() { w.seen = boundary - 1 }()
	s.writeTime()
	w.roll()
	if !s.started {
		s.started = true
		if !s.run(ctx, Startup, plan[Startup]) {
			return
		}
	}
	for set := First; set <= Last; set++ {
		if !s.run(ctx, set, plan[set]) {
			return
		}
	}
}

func (s *Sim) prepare() [setCount][]int {
	if s.fresh {
		return s.saved
	}
	var saved [setCount][]int
	for set := Startup; set < setCount; set++ {
		saved[set] = order(s.systems[set])
	}
	s.saved = saved
	s.fresh = true
	return saved
}

func (s *Sim) run(ctx context.Context, set Set, idxs []int) bool {
	for _, i := range idxs {
		if ctx.Err() != nil {
			return false
		}
		sys := &s.systems[set][i]
		w := s.World
		w.inSystem = true
		w.reader = sys.seq
		w.sysLast = sys.lastRun
		func() {
			defer func() {
				w.settle()
				sys.lastRun = w.clock
				w.clock++
				w.inSystem = false
				w.reader = outsideReader
				w.flush()
			}()
			sys.run(ctx, w)
		}()
	}
	return ctx.Err() == nil
}
