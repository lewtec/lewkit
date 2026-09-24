package gui

// Dirty is embedded in a [Model]. [See] and [Touch] return an updated flag.
// [Run] skips View while the flag is clear.
type Dirty struct{ dirty bool }

// Touch returns d marked changed.
func Touch(d Dirty) Dirty {
	d.dirty = true
	return d
}

// See returns next and a dirty flag. The flag is set when next differs from value.
func See[T comparable](d Dirty, value, next T) (Dirty, T) {
	if value == next {
		return d, value
	}
	d.dirty = true
	return d, next
}

// Consume reports a change since the last consume and clears the flag.
func (d *Dirty) Consume() bool {
	if d == nil || !d.dirty {
		return false
	}
	d.dirty = false
	return true
}
