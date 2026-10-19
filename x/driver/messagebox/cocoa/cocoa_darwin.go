//go:build darwin

package cocoa

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/internal/cocoaalert"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/prelude"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var errNotBound = errors.New("alert: thread not bound")

var (
	selAlloc   = objc.RegisterName("alloc")
	selInit    = objc.RegisterName("init")
	selMessage = objc.RegisterName("setMessageText:")
	selInfo    = objc.RegisterName("setInformativeText:")
	selStyle   = objc.RegisterName("setAlertStyle:")
	selButton  = objc.RegisterName("addButtonWithTitle:")
	selRun     = objc.RegisterName("runModal")
	selNew     = objc.RegisterName("new")
	selDrain   = objc.RegisterName("drain")
)

func available(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
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
	if err := cocoaalert.Open(); err != nil {
		return err
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	alert := objc.ID(objc.GetClass("NSAlert")).Send(selAlloc).Send(selInit)
	if alert == 0 {
		return cocoaalert.ErrAlert
	}
	alert.Send(selMessage, native.NSString(n.Title))
	alert.Send(selInfo, native.NSString(n.Message))
	alert.Send(selStyle, alertStyle(string(n.Style)))
	alert.Send(selButton, native.NSString("OK"))
	alert.Send(selRun)
	return nil
}
