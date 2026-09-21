// Package window opens a resizable host window backed by an *image.RGBA.
//
//	w, err := window.Open(ctx, window.Config{Title: "hi"})
//	pixels, err := window.Fit(tensor, w.Frame())
//	err = window.Present(ctx, tensor, evaluator, w.Frame())
//	err = w.Draw()
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (mem, cocoa, win32, x11).
package window

import (
	"context"
	"errors"
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	ErrClosed   = errors.New("window closed")
	ErrSize     = errors.New("invalid size")
	ErrNotBound = errors.New("thread not bound")
	ErrNotMain  = errors.New("not main thread")
	ErrInit     = errors.New("window init")
	ErrPresent  = errors.New("present")
)

const (
	defaultWidth  = 640
	defaultHeight = 480
	// DefaultFramePeriod is used when the host cannot read a display rate
	// and Config.Period is unset.
	DefaultFramePeriod = time.Second / 60
)

// Config is the initial window. Width or Height 0 means 640×480.
// Period 0 means the host display rate (or [DefaultFramePeriod]).
// The window is resizable after Open.
type Config struct {
	Title  string
	Width  int
	Height int
	Period time.Duration
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
// Size is the client size in pixels, without painting. Frame is the back
// buffer (Go RGBA, uint8, shape h×w×4). [Fit] resizes a tensor to that
// layout. Draw swaps it to the front (last swap wins) and the host blits
// on its next turn. Size may move during live resize; Draw still
// presents the last painted page. FramePeriod is the host display interval.
// Subscribe is an event source: Resize, Expose, Close, Pointer, Scroll, and Key.
// [Drive] is the immediate-mode paint loop; [Animate] is Drive that paints on tick and Resize.
// After Resize the next Frame has the new size.
type Window interface {
	Frame() *image.RGBA
	Front() *image.RGBA
	Size() image.Point
	FramePeriod() time.Duration
	Draw() error
	Resize(size image.Point) error
	Subscribe(ctx context.Context) <-chan Event
	Close() error
}

// Open asks the active window driver for a window.
func Open(ctx context.Context, cfg Config) (Window, error) {
	return driver.WithResult(ctx, func(d Driver) (Window, error) {
		return d.Open(ctx, cfg)
	})
}

// CloseWhenDone closes w when ctx is done. A nil ctx is ignored.
func CloseWhenDone(ctx context.Context, w Window) {
	if ctx == nil {
		return
	}
	context.AfterFunc(ctx, func() { _ = w.Close() })
}
