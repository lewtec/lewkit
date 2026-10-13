//go:build android

package android

import (
	"context"
	"image"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

func init() { driver.Register[window.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "window_android" }
func (factory) Name() string { return "Android surface" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.JavaVM()
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
	entry.HandleFontScale(func(scale float64) {
		w.SetFontScale(window.Scale(scale))
	})
	w.SetScale(displayScale())
	w.SetFontScale(displayFontScale())
	return w, nil
}

// displayScale reads DisplayMetrics.density. The surface is already physical
// pixels, so Open does not resize it. A JNI failure leaves the scale at one.
func displayScale() window.Scale {
	return resourceScale("getDisplayMetrics", "density")
}

// displayFontScale reads Configuration.fontScale, the system text-size
// preference. Density stays DisplayMetrics.density. A JNI failure leaves
// the preference at one.
func displayFontScale() window.Scale {
	return resourceScale("getConfiguration", "fontScale")
}

func resourceScale(method, field string) window.Scale {
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
	object, err := jni.AsRef(resources.Call(method))
	if err != nil || object == nil {
		return 1
	}
	defer object.Release()
	value, err := jni.Float(object.Field(field))
	if err != nil || value <= 0 {
		return 1
	}
	return window.Scale(value)
}

type win struct {
	*window.Buffer
	native uintptr
}

func (w *win) Draw() error { return w.Swap() }

func (w *win) Surface() window.Surface {
	return window.Surface{Kind: window.SurfaceAndroid, A: w.native}
}
