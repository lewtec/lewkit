//go:build darwin

package cocoa

import (
	"context"
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/prelude"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	watchOnce sync.Once
	watchErr  error
	observer  objc.ID
	bus       = event.New[daynight.Mode]()
	current   daynight.Mode
	mu        sync.Mutex

	selStandard  = objc.RegisterName("standardUserDefaults")
	selStringFor = objc.RegisterName("stringForKey:")
	selDefault   = objc.RegisterName("defaultCenter")
	selAdd       = objc.RegisterName("addObserver:selector:name:object:")
	selChanged   = objc.RegisterName("themeChanged:")
	selAlloc     = objc.RegisterName("alloc")
	selInit      = objc.RegisterName("init")
)

func available(context.Context) error {
	if err := frameworks(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func open(ctx context.Context) (daynight.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := startWatch(); err != nil {
		return nil, err
	}
	return host{}, nil
}

type host struct{}

func (host) Current(context.Context) (daynight.Mode, error) {
	var mode daynight.Mode
	onUI(func() {
		mu.Lock()
		current = readStyle()
		mode = current
		mu.Unlock()
	})
	return mode, nil
}

func (host) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	var scheme daynight.Mode
	onUI(func() {
		mu.Lock()
		scheme = readStyle()
		current = scheme
		mu.Unlock()
	})
	return daynight.Changes(ctx, scheme, bus.Subscribe(ctx)), nil
}

// onUI runs fn on the UI thread when one is bound. A command with no loop
// still reads defaults on the caller.
func onUI(fn func()) {
	if thread.Bound() && !thread.On() {
		thread.Do(fn)
		return
	}
	fn()
}

func frameworks() error {
	if _, err := native.Open("/System/Library/Frameworks/Foundation.framework/Foundation", native.Global|native.Lazy); err != nil {
		return err
	}
	_, err := native.Open("/System/Library/Frameworks/AppKit.framework/AppKit", native.Global|native.Lazy)
	return err
}

func startWatch() error {
	onUI(func() { watchOnce.Do(registerWatch) })
	return watchErr
}

func registerWatch() {
	if err := frameworks(); err != nil {
		watchErr = err
		return
	}
	class, err := objc.RegisterClass(
		"LewkitAppearanceObserver",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{{
			Cmd: selChanged,
			Fn: func(objc.ID, objc.SEL, objc.ID) {
				var mode daynight.Mode
				onUI(func() { mode = readStyle() })
				publish(mode)
			},
		}},
	)
	if err != nil {
		watchErr = err
		return
	}
	observer = objc.ID(class).Send(selAlloc).Send(selInit)
	if observer == 0 {
		watchErr = fmt.Errorf("%w: observer", driver.ErrUnavailable)
		return
	}
	center := objc.ID(objc.GetClass("NSDistributedNotificationCenter")).Send(selDefault)
	center.Send(selAdd, observer, selChanged, native.NSString("AppleInterfaceThemeChangedNotification"), objc.ID(0))
	mu.Lock()
	current = readStyle()
	mu.Unlock()
}

func publish(scheme daynight.Mode) {
	mu.Lock()
	same := current == scheme
	current = scheme
	mu.Unlock()
	if !same {
		bus.Publish(scheme)
	}
}

func readStyle() daynight.Mode {
	defaults := objc.ID(objc.GetClass("NSUserDefaults")).Send(selStandard)
	if defaults == 0 {
		return daynight.Light
	}
	value := defaults.Send(selStringFor, native.NSString("AppleInterfaceStyle"))
	if value == 0 {
		return daynight.Light
	}
	text := native.UTF8String(value)
	return fromInterfaceStyle(text)
}
