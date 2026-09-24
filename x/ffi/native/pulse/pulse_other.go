//go:build !linux

package pulse

import "context"

// Available reports that this OS build has no Pulse binding.
func Available() error {
	return errUnavailable
}

// List returns no sinks on this OS.
func List(context.Context) ([]Sink, error) {
	return nil, Available()
}

// Stream is unused outside Linux.
type Stream struct{}

// Playback is unavailable outside Linux.
func Playback(context.Context, string, string, string, Sample, int, int) (*Stream, error) {
	return nil, Available()
}

// Write is unavailable outside Linux.
func (s *Stream) Write([]byte) error { return Available() }

// Close is unavailable outside Linux.
func (s *Stream) Close() error { return Available() }
