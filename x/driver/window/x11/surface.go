package x11

import (
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
)

func (w *xwin) Surface() window.Surface {
	if w == nil {
		return window.Surface{}
	}
	return window.Surface{Kind: window.SurfaceX11, A: xlibDisplay(), B: uintptr(w.wid)}
}

var (
	xlibDpy uintptr
	xOpen   func(name *byte) uintptr
	loadX11 = native.Once(func() error {
		lib, err := native.OpenChain(native.Global|native.Lazy, "libX11.so.6")
		if err != nil {
			return err
		}
		if err := native.Bind(lib, "XOpenDisplay", &xOpen); err != nil {
			return err
		}
		if xOpen != nil {
			xlibDpy = xOpen(nil)
		}
		return nil
	})
)

func xlibDisplay() uintptr {
	if loadX11() != nil {
		return 0
	}
	return xlibDpy
}
