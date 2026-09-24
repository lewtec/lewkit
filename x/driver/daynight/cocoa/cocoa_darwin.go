//go:build darwin

package cocoa

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
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

	selStandard   = objc.RegisterName("standardUserDefaults")
	selStringFor  = objc.RegisterName("stringForKey:")
	selUTF8       = objc.RegisterName("stringWithUTF8String:")
	selUTF8String = objc.RegisterName("UTF8String")
	selDefault    = objc.RegisterName("defaultCenter")
	selAdd        = objc.RegisterName("addObserver:selector:name:object:")
	selChanged    = objc.RegisterName("themeChanged:")
	selAlloc      = objc.RegisterName("alloc")
	selInit       = objc.RegisterName("init")
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
	mu.Lock()
	defer mu.Unlock()
	current = readStyle()
	return current, nil
}

func (host) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	mu.Lock()
	scheme := readStyle()
	current = scheme
	mu.Unlock()
	return daynight.Changes(ctx, scheme, bus.Subscribe(ctx)), nil
}

func frameworks() error {
	if _, err := native.Open("/System/Library/Frameworks/Foundation.framework/Foundation", native.Global|native.Lazy); err != nil {
		return err
	}
	_, err := native.Open("/System/Library/Frameworks/AppKit.framework/AppKit", native.Global|native.Lazy)
	return err
}

func startWatch() error {
	watchOnce.Do(func() {
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
					publish(readStyle())
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
		center.Send(selAdd, observer, selChanged, nsString("AppleInterfaceThemeChangedNotification"), objc.ID(0))
		mu.Lock()
		current = readStyle()
		mu.Unlock()
	})
	return watchErr
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
	value := defaults.Send(selStringFor, nsString("AppleInterfaceStyle"))
	if value == 0 {
		return daynight.Light
	}
	text := cocoaString(value)
	return fromInterfaceStyle(text)
}

func nsString(text string) objc.ID {
	bytes := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selUTF8, unsafe.Pointer(&bytes[0]))
}

func cocoaString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	pointer := id.Send(selUTF8String)
	if pointer == 0 {
		return ""
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), 1<<20)
	for i, b := range raw {
		if b == 0 {
			return string(raw[:i])
		}
	}
	return ""
}
