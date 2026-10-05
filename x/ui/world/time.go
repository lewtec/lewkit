package world

import "context"

// Time is the clock for one [Sim]. [Sim.Step] writes it at the start of
// the frame, after the tick advances, so a system reads this frame's
// delta. Delta and Elapsed are seconds. This is not a fixed timestep.
type Time struct {
	Delta   float64
	Elapsed float64
}

// Step stores delta on [Time] and runs one [Sim.Frame]. A negative delta
// is zero. The host already measured the duration. The world does not
// read a window tick.
func (s *Sim) Step(ctx context.Context, delta float64) {
	if s == nil {
		return
	}
	if delta < 0 {
		delta = 0
	}
	s.delta = delta
	s.stepped = true
	s.Frame(ctx)
}

func (s *Sim) writeTime() {
	if s == nil || !s.stepped {
		return
	}
	s.stepped = false
	clock, _ := Read[Time](s.World)
	clock.Delta = s.delta
	clock.Elapsed += s.delta
	Put(s.World, clock)
}
