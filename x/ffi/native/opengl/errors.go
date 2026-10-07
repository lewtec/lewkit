package opengl

import "errors"

const (
	// maxDim keeps a frame inside a size every GL 3.3 / ES 3.0 target accepts.
	maxDim = 8192
	// instanceStride is one rounded rect: 16 little-endian float32 values.
	instanceStride = 64
)

var (
	// ErrClosed means Draw or Adopt ran after Close.
	ErrClosed = errors.New("opengl closed")
	// ErrLost means the native surface is gone.
	ErrLost = errors.New("opengl surface lost")
	// ErrSize means the frame size or the instance bytes are not usable.
	ErrSize = errors.New("opengl size")
	// ErrUnavailable means OpenGL cannot be opened here.
	ErrUnavailable = errors.New("opengl unavailable")
)
