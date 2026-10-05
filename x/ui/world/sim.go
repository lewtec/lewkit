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

// Frame clears the previous message log, runs Startup once, then runs
// First through Last. A canceled ctx runs no further systems. The
// system that observed the cancel still flushes its deferred edits.
func (s *Sim) Frame(ctx context.Context) {
	if s == nil || s.World == nil {
		if s != nil {
			s.stepped = false
		}
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		s.stepped = false
		return
	}
	s.World.tick++
	s.writeTime()
	s.World.roll()
	if !s.started {
		s.started = true
		if !s.run(ctx, Startup) {
			return
		}
	}
	for set := First; set <= Last; set++ {
		if !s.run(ctx, set) {
			return
		}
	}
}

func (s *Sim) run(ctx context.Context, set Set) bool {
	systems := order(s.systems[set])
	for _, sys := range systems {
		if ctx.Err() != nil {
			return false
		}
		s.World.inSystem = true
		func() {
			defer func() {
				s.World.inSystem = false
				s.World.flush()
			}()
			sys(ctx, s.World)
		}()
	}
	return ctx.Err() == nil
}
