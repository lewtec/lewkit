package x11

import (
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
)

func (w *xwin) Surface() window.Surface {
	if w == nil {
		return window.Surface{}
	}
	dpy, _ := xlibDisplay()
	return window.Surface{Kind: window.SurfaceX11, A: dpy, B: uintptr(w.wid)}
}

// xlibDisplay opens libX11 and returns the display from that same load.
var xlibDisplay = native.Singleton(func() (uintptr, error) {
	xlib, err := native.Bind[Xlib](native.Global|native.Lazy, "libX11.so.6")
	if err != nil {
		return 0, err
	}
	return xlib.XOpenDisplay(nil), nil
})
