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
	xlib    Xlib
	xlibDpy uintptr
	loadX11 = native.Once(func() error {
		var err error
		xlib, err = native.Bind[Xlib](native.Global|native.Lazy, "libX11.so.6")
		if err != nil {
			return err
		}
		if xlib.XOpenDisplay != nil {
			xlibDpy = xlib.XOpenDisplay(nil)
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
