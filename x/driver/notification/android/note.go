package android

import "math"

const (
	posted  = 0
	waiting = 1
	denied  = 2
)

func percentOf(progress float64, has bool) int {
	if !has {
		return 0
	}
	p := int(math.Round(progress * 100))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}
