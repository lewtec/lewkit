// Package period sizes one native audio playback period.
package period

// Bytes is one 20 ms period. The count is at least one frame and a whole number of frames.
func Bytes(rate, frame int) int {
	n := rate * frame / 50
	if n < frame {
		n = frame
	}
	return n - n%frame
}
