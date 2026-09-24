//go:build !darwin

package coreaudio

import "context"

// Available reports that this OS build has no CoreAudio binding.
func Available() error {
	return errUnavailable
}

// Devices returns no endpoints on this OS.
func Devices() ([]Device, error) {
	return nil, Available()
}

// Stream is unused outside macOS.
type Stream struct{}

// Open is unavailable outside macOS.
func Open(context.Context, string, Layout) (*Stream, error) {
	return nil, Available()
}

// Write is unavailable outside macOS.
func (s *Stream) Write([]byte) error { return Available() }

// Close is unavailable outside macOS.
func (s *Stream) Close() error { return Available() }
