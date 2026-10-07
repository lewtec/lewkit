// Package present paints one GUI frame onto a host surface the caller owns.
//
// The frame is the draw list gui already lowers: an optional tightly packed
// RGBA8 underlay, rounded-rect instances, then tightly packed RGBA8 glyph ink.
// [Open] asks the registered drivers, highest weight first, and uses the
// first screen that accepts the surface. Metal is the Apple screen, so the
// process does not need MoltenVK. Vulkan remains the screen when libvulkan
// is the one that can attach. OpenGL is the fallback after Vulkan on
// Windows, Linux, and Android. Apple does not open it.
package present

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/ndarray"
)

const (
	// InstanceStride is the bytes of one rounded rect: 16 little-endian float32.
	// The values are box (x, y, width, height), color (red, green, blue, alpha
	// in 0..255), radius and clip (radius, clip x, clip y, clip width), then
	// clip height and three zeros.
	InstanceStride = 64
)

var (
	// ErrLost means the native view is gone until the host creates another one.
	ErrLost = errors.New("present surface lost")
	// ErrSize means the frame size or the instance bytes are not usable.
	ErrSize = errors.New("present size")
	// ErrClosed means the screen has been closed.
	ErrClosed = errors.New("present closed")
)

// Screen paints under, then fills, then ink, and presents.
type Screen interface {
	Draw(ctx context.Context, instances, under, ink []byte, width, height int) error
	// Adopt points the screen at a replacement native view and size.
	Adopt(window uintptr, width, height int) error
	Close() error
}

// Driver attaches a screen to a host surface.
// kind, a, and b are the [window.Surface] fields. The driver does not close
// that view.
type Driver interface {
	Open(ctx context.Context, kind int, a, b uintptr, width, height int) (Screen, error)
}

// Painter is a screen that can present a mounted (height, width, 4) tensor
// on its own device. Metal does not implement it; the draw list stays the frame.
type Painter interface {
	Screen
	Paint(ctx context.Context, pixels *ndarray.Tensor[uint8]) error
	Evaluator() ndarray.Evaluator
}
