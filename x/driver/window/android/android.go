//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"image"

	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[window.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "window_android" }
func (factory) Name() string { return "Android surface" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (window.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	native, width, height, err := entry.RequestSurface(ctx)
	if err != nil {
		return nil, err
	}
	buf := window.NewBuffer(width, height)
	buf.SetFramePeriod(cfg.Period)
	w := &win{Buffer: buf, native: native}
	entry.HandlePointer(func(x, y, action int) {
		pos := image.Pt(x, y)
		switch action {
		case 0:
			w.Emit(window.Pointer{Pos: pos, Button: 1, Pressed: true, Buttons: window.ButtonLeft})
		case 1:
			w.Emit(window.Pointer{Pos: pos, Button: 1, Pressed: false})
		default:
			w.Emit(window.Pointer{Pos: pos})
		}
	})
	entry.HandleResize(func(width, height int) {
		_ = w.Resize(image.Pt(width, height))
	})
	entry.HandleSurface(func(ptr uintptr, width, height int) {
		w.native = ptr
		slog.Info("android surface", "width", width, "height", height)
		_ = w.Resize(image.Pt(width, height))
	})
	entry.HandleSurfaceLost(func() {
		w.native = 0
		slog.Info("android surface", "lost", true)
	})
	return w, nil
}

type win struct {
	*window.Buffer
	native uintptr
}

func (w *win) Draw() error { return w.Swap() }

func (w *win) Surface() window.Surface {
	return window.Surface{Kind: window.SurfaceAndroid, A: w.native}
}
