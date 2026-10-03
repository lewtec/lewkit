//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[brightness.Driver](factory{}) }

var (
	errNoBrightness = errors.New("screen brightness is unset")
	errNoWindow     = errors.New("no foreground window")
	errWindow       = errors.New("window brightness")
)

type factory struct{}

func (factory) ID() string   { return "brightness_android" }
func (factory) Name() string { return "Android brightness" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (brightness.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) SetBrightness(ctx context.Context, level float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if level < 0 || level > 1 {
		return fmt.Errorf("%w: %v", errScale, level)
	}
	// The post returns before the main looper runs. The wait stays on this
	// goroutine so the loader thread can serve the Runnable's Java calls.
	// The proxy stays alive until that run finishes.
	done := make(chan error, 1)
	held, err := postBrightness(float32(level), done)
	if err != nil {
		held.release()
		return err
	}
	defer held.release()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (backend) Status(ctx context.Context) (*brightness.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	level, err := currentBrightness()
	if err != nil {
		return nil, err
	}
	return &brightness.Device{Name: "display", Brightness: level}, nil
}

type heldRefs struct {
	looper  *jni.Ref
	handler *jni.Ref
	proxy   *jni.Ref
}

func (h heldRefs) release() {
	if h.proxy != nil {
		h.proxy.Release()
	}
	if h.handler != nil {
		h.handler.Release()
	}
	if h.looper != nil {
		h.looper.Release()
	}
}

func postBrightness(level float32, done chan<- error) (heldRefs, error) {
	var held heldRefs
	looper, err := host.Ref(jni.CallStatic("android.os.Looper", "getMainLooper"))
	if err != nil {
		return held, err
	}
	if looper == nil {
		return held, errNoWindow
	}
	held.looper = looper
	handler, err := jni.New("android.os.Handler", looper)
	if err != nil {
		return held, err
	}
	held.handler = handler
	proxy, err := jni.Proxy("java.lang.Runnable", func(method string, _ []any) (any, error) {
		if method != "run" {
			return nil, nil
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- fmt.Errorf("%w: %v", errWindow, recovered)
			}
		}()
		done <- applyWindowBrightness(level)
		return nil, nil
	})
	if err != nil {
		return held, err
	}
	held.proxy = proxy
	_, err = handler.Call("post", proxy)
	if err != nil {
		return held, err
	}
	return held, nil
}

func applyWindowBrightness(level float32) error {
	fg, err := host.Ref(jni.StaticField("lewkit.Host", "foreground"))
	if err != nil {
		return err
	}
	if fg == nil {
		return errNoWindow
	}
	defer fg.Release()
	window, err := host.Ref(fg.Call("getWindow"))
	if err != nil {
		return err
	}
	if window == nil {
		return errNoWindow
	}
	defer window.Release()
	attrs, err := host.Ref(window.Call("getAttributes"))
	if err != nil {
		return err
	}
	if attrs == nil {
		return errNoWindow
	}
	defer attrs.Release()
	class, err := host.Ref(attrs.Call("getClass"))
	if err != nil {
		return err
	}
	if class == nil {
		return errNoWindow
	}
	defer class.Release()
	field, err := host.Ref(class.Call("getField", "screenBrightness"))
	if err != nil {
		return err
	}
	if field == nil {
		return errNoBrightness
	}
	defer field.Release()
	if _, err = field.Call("setFloat", attrs, level); err != nil {
		return err
	}
	_, err = window.Call("setAttributes", attrs)
	return err
}

func currentBrightness() (float64, error) {
	fg, err := host.Ref(jni.StaticField("lewkit.Host", "foreground"))
	if err != nil {
		return 0, err
	}
	if fg != nil {
		defer fg.Release()
		level, ok, err := windowLevel(fg)
		if err != nil || ok {
			return level, err
		}
	}
	return systemBrightness()
}

func windowLevel(fg *jni.Ref) (float64, bool, error) {
	window, err := host.Ref(fg.Call("getWindow"))
	if err != nil || window == nil {
		return 0, false, err
	}
	defer window.Release()
	attrs, err := host.Ref(window.Call("getAttributes"))
	if err != nil || attrs == nil {
		return 0, false, err
	}
	defer attrs.Release()
	level, err := host.Float(attrs.Field("screenBrightness"))
	if err != nil {
		return 0, false, err
	}
	if level < 0 || level > 1 {
		return 0, false, nil
	}
	return level, true, nil
}

func systemBrightness() (float64, error) {
	app, err := host.Context()
	if err != nil {
		return 0, err
	}
	defer app.Release()
	resolver, err := host.Ref(app.Call("getContentResolver"))
	if err != nil {
		return 0, err
	}
	if resolver == nil {
		return 0, errNoBrightness
	}
	defer resolver.Release()
	resources, err := host.Ref(app.Call("getResources"))
	if err != nil {
		return 0, err
	}
	if resources == nil {
		return 0, errNoBrightness
	}
	defer resources.Release()
	low, err := resourceInt(resources, "config_screenBrightnessSettingMinimum")
	if err != nil {
		return 0, err
	}
	high, err := resourceInt(resources, "config_screenBrightnessSettingMaximum")
	if err != nil {
		return 0, err
	}
	if high == 0 {
		low, high = 0, 255
	}
	key, err := host.Text(jni.StaticField("android.provider.Settings$System", "SCREEN_BRIGHTNESS"))
	if err != nil {
		return 0, err
	}
	raw, err := host.Int(jni.CallStatic("android.provider.Settings$System", "getInt", resolver, key, -1))
	if err != nil {
		return 0, err
	}
	if raw < 0 {
		return 0, errNoBrightness
	}
	return brightnessFraction(raw, low, high)
}

func resourceInt(res *jni.Ref, name string) (int, error) {
	id, err := host.Int(res.Call("getIdentifier", name, "integer", "android"))
	if err != nil || id == 0 {
		return 0, err
	}
	return host.Int(res.Call("getInteger", id))
}
