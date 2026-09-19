package event

import "time"

// FPS is a smoothed frames-per-second reading. Get records the instant
// since the previous Get and returns the EMA (alpha 0.15).
type FPS struct {
	last  time.Time
	value float64
}

// Get records now, updates the smoothed rate, and returns it.
func (f *FPS) Get() float64 {
	if f == nil {
		return 0
	}
	now := time.Now()
	if !f.last.IsZero() {
		dt := now.Sub(f.last).Seconds()
		if dt > 0 {
			inst := 1 / dt
			if f.value == 0 {
				f.value = inst
			} else {
				f.value = f.value*0.85 + inst*0.15
			}
		}
	}
	f.last = now
	return f.value
}
