package x11

import (
	"sync"

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
	xlibOnce sync.Once
	xlibDpy  uintptr
	xOpen    func(name *byte) uintptr
)

func xlibDisplay() uintptr {
	xlibOnce.Do(func() {
		lib, err := native.Open("libX11.so.6", native.Global|native.Lazy)
		if err != nil {
			return
		}
		native.Func(lib, "XOpenDisplay", &xOpen)
		if xOpen != nil {
			xlibDpy = xOpen(nil)
		}
	})
	return xlibDpy
}
