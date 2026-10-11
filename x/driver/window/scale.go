package window

import "math"

// Scale is device pixels per logical pixel. One is the baseline.
// Two means each authored pixel covers two device pixels.
type Scale float64

// Factor is the scale, or one when the value is missing or not finite.
func (s Scale) Factor() float64 {
	f := float64(s)
	if f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 1
	}
	return f
}

// Device converts a logical length to device pixels, rounded.
// A positive length is at least one device pixel. Zero stays zero.
func (s Scale) Device(logical int) int {
	if logical == 0 {
		return 0
	}
	n := int(math.Round(float64(logical) * s.Factor()))
	if logical > 0 && n < 1 {
		return 1
	}
	if logical < 0 && n > -1 {
		return -1
	}
	return n
}

// Logical converts device pixels to a logical length, rounded.
func (s Scale) Logical(device int) int {
	if device == 0 {
		return 0
	}
	n := int(math.Round(float64(device) / s.Factor()))
	if device > 0 && n < 1 {
		return 1
	}
	if device < 0 && n > -1 {
		return -1
	}
	return n
}
