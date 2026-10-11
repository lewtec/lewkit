//go:build android

package android

import (
	"context"
	"fmt"
	"image"

	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ffi/jni"
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
	entry.HandleInsets(func(left, top, right, bottom, width, height int) {
		w.SetDead(window.InsetZones(width, height, left, top, right, bottom))
	})
	entry.HandlePointer(func(x, y, action int) {
		w.Emit(window.TouchPointer(x, y, action))
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
	w.SetScale(displayScale())
	return w, nil
}

// displayScale reads DisplayMetrics.density. The surface is already physical
// pixels, so Open does not resize it. A JNI failure leaves the scale at one.
func displayScale() window.Scale {
	app, err := jni.Context()
	if err != nil || app == nil {
		return 1
	}
	defer app.Release()
	resources, err := jni.AsRef(app.Call("getResources"))
	if err != nil || resources == nil {
		return 1
	}
	defer resources.Release()
	metrics, err := jni.AsRef(resources.Call("getDisplayMetrics"))
	if err != nil || metrics == nil {
		return 1
	}
	defer metrics.Release()
	density, err := jni.Float(metrics.Field("density"))
	if err != nil || density <= 0 {
		return 1
	}
	return window.Scale(density)
}

type win struct {
	*window.Buffer
	native uintptr
}

func (w *win) Draw() error { return w.Swap() }

func (w *win) Surface() window.Surface {
	return window.Surface{Kind: window.SurfaceAndroid, A: w.native}
}
