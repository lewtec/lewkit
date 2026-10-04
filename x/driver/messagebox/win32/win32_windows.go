//go:build windows

package win32

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/messagebox"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	mbOK              = 0
	mbIconError       = 0x10
	mbIconWarning     = 0x30
	mbIconInformation = 0x40
)

var procMessageBox = native.ProcOf("user32.dll", "MessageBoxW")

func available(context.Context) error { return nil }

func open(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	text, err := syscall.UTF16PtrFromString(n.Message)
	if err != nil {
		return err
	}
	caption, err := syscall.UTF16PtrFromString(n.Title)
	if err != nil {
		return err
	}
	r, _, callErr := procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(caption)), mbOK|iconOf(n.Style))
	if r == 0 {
		return callErr
	}
	return nil
}

func iconOf(style string) uintptr {
	switch messagebox.NormalizeStyle(style) {
	case messagebox.StyleWarning:
		return mbIconWarning
	case messagebox.StyleCritical:
		return mbIconError
	default:
		return mbIconInformation
	}
}
