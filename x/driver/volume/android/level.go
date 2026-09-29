package android

import "math"

const musicStream int32 = 3

func fraction(cur, max int32) float64 {
	if max <= 0 {
		return 0
	}
	return float64(cur) / float64(max)
}

func index(level float64, max int32) int32 {
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	if max <= 0 {
		return 0
	}
	return int32(math.Round(level * float64(max)))
}
