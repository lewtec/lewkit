//go:build darwin

package cocoa

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	cocoaFramework = "/System/Library/Frameworks/Cocoa.framework/Cocoa"
)

var (
	errAlert    = errors.New("alert")
	errNotBound = errors.New("alert: thread not bound")

	appOnce sync.Once
	appErr  error
)

var (
	selAlloc   = objc.RegisterName("alloc")
	selInit    = objc.RegisterName("init")
	selMessage = objc.RegisterName("setMessageText:")
	selInfo    = objc.RegisterName("setInformativeText:")
	selStyle   = objc.RegisterName("setAlertStyle:")
	selButton  = objc.RegisterName("addButtonWithTitle:")
	selRun     = objc.RegisterName("runModal")
	selUTF8    = objc.RegisterName("stringWithUTF8String:")
	selNew     = objc.RegisterName("new")
	selDrain   = objc.RegisterName("drain")
)

func available(context.Context) error {
	if driver.AppMode() {
		return fmt.Errorf("%w: app mode", driver.ErrIncompatible)
	}
	return nil
}

func open(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !thread.Bound() {
		return errNotBound
	}
	var err error
	thread.Do(func() {
		err = show(n)
	})
	return err
}

func show(n messagebox.Notice) error {
	if err := ensureApp(); err != nil {
		return err
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	alert := objc.ID(objc.GetClass("NSAlert")).Send(selAlloc).Send(selInit)
	if alert == 0 {
		return errAlert
	}
	alert.Send(selMessage, nsString(n.Title))
	alert.Send(selInfo, nsString(n.Message))
	alert.Send(selStyle, alertStyle(n.Style))
	alert.Send(selButton, nsString("OK"))
	alert.Send(selRun)
	return nil
}

func ensureApp() error {
	appOnce.Do(func() {
		if _, err := native.Open(cocoaFramework, native.Global|native.Lazy); err != nil {
			appErr = fmt.Errorf("%w: %w", errAlert, err)
		}
	})
	return appErr
}

func nsString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selUTF8, unsafe.Pointer(&raw[0]))
}
