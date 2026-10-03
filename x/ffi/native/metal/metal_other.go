//go:build !darwin

package metal

import "errors"

var (
	// ErrClosed means Draw or Adopt ran after Close.
	ErrClosed = errors.New("metal closed")
	// ErrLost means the view is gone.
	ErrLost = errors.New("metal surface lost")
	// ErrSize means the frame size or instance bytes are not usable.
	ErrSize = errors.New("metal size")
	// ErrUnavailable means Metal cannot be opened here.
	ErrUnavailable = errors.New("metal unavailable")
)

// Screen is the Metal drawable for one view. This build has no Metal.
type Screen struct{}

// SetUI records the function that runs on the platform UI thread.
// This build keeps it unused.
func SetUI(func(func())) {}

// Available reports that Metal is not on this OS.
func Available() error { return ErrUnavailable }

// OpenNative reports that Metal is not on this OS.
func OpenNative(int, uintptr, int, int) (*Screen, error) {
	return nil, ErrUnavailable
}

// Draw reports that the screen is closed.
func (*Screen) Draw([]byte, []byte, []byte, int, int) error { return ErrUnavailable }

// Adopt reports that the screen is closed.
func (*Screen) Adopt(uintptr, int, int) error { return ErrUnavailable }

// Close is a no-op.
func (*Screen) Close() error { return nil }
