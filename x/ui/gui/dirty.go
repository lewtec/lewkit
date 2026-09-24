package gui

// Dirty is embedded in a [Model]. [Set] and [Dirty.Mark] record that
// Update changed state. [Run] skips View while the flag is clear.
type Dirty struct{ dirty bool }

// Mark records that Update changed the model.
func (d *Dirty) Mark() {
	if d != nil {
		d.dirty = true
	}
}

// Consume reports a change since the last consume and clears the flag.
func (d *Dirty) Consume() bool {
	if d == nil || !d.dirty {
		return false
	}
	d.dirty = false
	return true
}

// Set assigns dst and marks d when the value changes.
func Set[T comparable](d *Dirty, dst *T, src T) {
	if d == nil || dst == nil || *dst == src {
		return
	}
	*dst = src
	d.Mark()
}
