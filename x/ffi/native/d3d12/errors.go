package d3d12

import "errors"

var (
	// ErrClosed means Draw, Adopt, or a compute call ran after Close.
	ErrClosed = errors.New("d3d12 closed")
	// ErrLost means the window is gone or the device was removed.
	ErrLost = errors.New("d3d12 surface lost")
	// ErrSize means the frame size or instance bytes are not usable.
	ErrSize = errors.New("d3d12 size")
	// ErrUnavailable means Direct3D 12 cannot be opened here.
	ErrUnavailable = errors.New("d3d12 unavailable")
)
