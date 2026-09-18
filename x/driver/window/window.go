// Package window opens a resizable host window backed by an *image.RGBA.
//
//	w, err := window.Open(ctx, window.Config{Title: "hi"})
//	draw.Draw(w.Frame(), w.Frame().Bounds(), src, src.Bounds().Min, draw.Src)
//	err = w.Draw()
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (mem, cocoa, win32, x11).
package window

import (
	"context"
	"errors"
	"image"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	ErrClosed = errors.New("window closed")
	ErrSize   = errors.New("invalid size")
)

const (
	defaultWidth  = 640
	defaultHeight = 480
)

// Config is the initial window. Width or Height 0 means 640×480.
// The window is resizable after Open.
type Config struct {
	Title  string
	Width  int
	Height int
}

// Size is the initial client size. Width or Height 0 becomes 640 or 480.
func (c Config) Size() (int, int, error) {
	if c.Width < 0 || c.Height < 0 {
		return 0, 0, ErrSize
	}
	w, h := c.Width, c.Height
	if w == 0 {
		w = defaultWidth
	}
	if h == 0 {
		h = defaultHeight
	}
	return w, h, nil
}

// Driver opens a host window.
type Driver interface {
	Open(ctx context.Context, cfg Config) (Window, error)
}

// Window is a resizable host window backed by an *image.RGBA.
//
// Frame is the current buffer; draw into it with [image/draw.Draw].
// Draw copies that buffer onto the host surface.
// After Resize, or after the user resizes the window, the next Frame
// has the new size.
type Window interface {
	Frame() *image.RGBA
	Draw() error
	Resize(size image.Point) error
	Close() error
}

// Open asks the active window driver for a window.
func Open(ctx context.Context, cfg Config) (Window, error) {
	return driver.WithResult(ctx, func(d Driver) (Window, error) {
		return d.Open(ctx, cfg)
	})
}
