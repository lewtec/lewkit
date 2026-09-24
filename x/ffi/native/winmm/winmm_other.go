//go:build !windows

package winmm

import "context"

// Available reports that this OS build has no waveOut binding.
func Available() error {
	return errUnavailable
}

// Devices returns no endpoints on this OS.
func Devices() ([]Device, error) {
	return nil, Available()
}

// Stream is unused outside Windows.
type Stream struct{}

// Open is unavailable outside Windows.
func Open(context.Context, string, Layout) (*Stream, error) {
	return nil, Available()
}

// Write is unavailable outside Windows.
func (s *Stream) Write([]byte) error { return Available() }

// Close is unavailable outside Windows.
func (s *Stream) Close() error { return Available() }
