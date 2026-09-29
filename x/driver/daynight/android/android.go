//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[daynight.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "daynight_android" }
func (factory) Name() string { return "Android night mode" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(ctx context.Context) (daynight.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := startWatch(); err != nil {
		return nil, err
	}
	return backend{}, nil
}

type backend struct{}

var (
	mu      sync.Mutex
	current daynight.Mode
	known   bool
	bus     = event.New[daynight.Mode]()

	watchOnce sync.Once
	watchErr  error
	proxy     *jni.Ref
)

func (backend) Current(ctx context.Context) (daynight.Mode, error) {
	if err := ctx.Err(); err != nil {
		return daynight.Light, err
	}
	return read()
}

func (backend) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scheme, err := read()
	if err != nil {
		return nil, err
	}
	return daynight.Changes(ctx, scheme, bus.Subscribe(ctx)), nil
}

func read() (daynight.Mode, error) {
	app, err := host.Context()
	if err != nil {
		return daynight.Light, err
	}
	defer app.Release()
	resources, err := host.Ref(app.Call("getResources"))
	if err != nil {
		return daynight.Light, err
	}
	if resources == nil {
		return daynight.Light, fmt.Errorf("%w: resources", driver.ErrUnavailable)
	}
	defer resources.Release()
	cfg, err := host.Ref(resources.Call("getConfiguration"))
	if err != nil {
		return daynight.Light, err
	}
	if cfg == nil {
		return daynight.Light, fmt.Errorf("%w: configuration", driver.ErrUnavailable)
	}
	defer cfg.Release()
	scheme, err := modeOf(cfg)
	if err != nil {
		return daynight.Light, err
	}
	mu.Lock()
	current = scheme
	known = true
	mu.Unlock()
	return scheme, nil
}

func modeOf(cfg *jni.Ref) (daynight.Mode, error) {
	mask, err := host.Int(jni.StaticField("android.content.res.Configuration", "UI_MODE_NIGHT_MASK"))
	if err != nil {
		return daynight.Light, err
	}
	yes, err := host.Int(jni.StaticField("android.content.res.Configuration", "UI_MODE_NIGHT_YES"))
	if err != nil {
		return daynight.Light, err
	}
	uiMode, err := host.Int(cfg.Field("uiMode"))
	if err != nil {
		return daynight.Light, err
	}
	return modeFromUI(uiMode, mask, yes), nil
}

func startWatch() error {
	watchOnce.Do(func() {
		app, err := host.Context()
		if err != nil {
			watchErr = err
			return
		}
		defer app.Release()
		callback, err := jni.Proxy("android.content.ComponentCallbacks", onConfiguration)
		if err != nil {
			watchErr = err
			return
		}
		if _, err = app.Call("registerComponentCallbacks", callback); err != nil {
			callback.Release()
			watchErr = err
			return
		}
		proxy = callback
	})
	return watchErr
}

func onConfiguration(method string, args []any) (any, error) {
	defer releaseArgs(args)
	if method != "onConfigurationChanged" || len(args) == 0 {
		return nil, nil
	}
	cfg, ok := args[0].(*jni.Ref)
	if !ok || cfg == nil {
		return nil, nil
	}
	scheme, err := modeOf(cfg)
	if err != nil {
		slog.Warn("android night mode", "err", err)
		return nil, nil
	}
	publish(scheme)
	return nil, nil
}

func releaseArgs(args []any) {
	for _, arg := range args {
		if ref, ok := arg.(*jni.Ref); ok {
			ref.Release()
		}
	}
}

func publish(scheme daynight.Mode) {
	mu.Lock()
	same := known && current == scheme
	current = scheme
	known = true
	mu.Unlock()
	if !same {
		bus.Publish(scheme)
	}
}
