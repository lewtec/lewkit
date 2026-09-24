package gui

// Dirty is embedded in a [Model]. [Set] and [Dirty.MarkDirty] record that
// Update changed state. [Run] skips View while the flag is clear.
type Dirty struct{ dirty bool }

// MarkDirty records that Update changed the model.
func (d *Dirty) MarkDirty() {
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

// Set assigns dst and marks the model when the value changes.
// M is the [Model] that embeds [Dirty].
func Set[M interface {
	Model
	MarkDirty()
}, T comparable](model M, dst *T, src T) {
	if dst == nil || *dst == src {
		return
	}
	*dst = src
	model.MarkDirty()
}
