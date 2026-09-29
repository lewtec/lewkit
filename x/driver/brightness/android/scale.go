package android

import "errors"

var errScale = errors.New("screen brightness scale")

// brightnessFraction maps a Settings.System integer onto 0..1.
func brightnessFraction(value, low, high int) (float64, error) {
	if high <= low {
		return 0, errScale
	}
	if value < low {
		value = low
	}
	if value > high {
		value = high
	}
	return float64(value-low) / float64(high-low), nil
}
