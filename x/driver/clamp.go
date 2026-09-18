package driver

// Clamp01 clamps v to [0, 1].
func Clamp01(v float64) float64 {
	return min(1, max(0, v))
}
