//go:build linux && !android

package vulkan

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	ErrXlibHostCreateXlibSurface = errors.New("xlib surface")
)

const extHostSurface = "VK_KHR_xlib_surface"

func surfaceExtensions() []string { return []string{extHostSurface} }

type xlibSurfaceInfo struct {
	sType  int32
	pNext  uintptr
	flags  uint32
	dpy    uintptr
	window uint64
}

type xlibHost struct {
	screen     *Screen
	dpy        uintptr
	win        uint64
	borrowed   bool
	createXlib func(inst uintptr, info *xlibSurfaceInfo, alloc uintptr, surface *uint64) int32
}

func attachHost(s *Screen, kind int, a, b uintptr) (hostSurface, error) {
	if kind != 1 || a == 0 || b == 0 {
		return nil, ErrUnavailable
	}
	return &xlibHost{screen: s, dpy: a, win: uint64(b), borrowed: true}, nil
}

func loadHost(w *wsi, d *Device) error {
	return nil
}

func (h *xlibHost) poll() {
	if h == nil || h.dpy == 0 {
		return
	}
	x11, err := loadX11()
	if err != nil {
		return
	}
	for x11.XPending(h.dpy) > 0 {
		var raw [192]byte
		ev := (*xEvent)(unsafe.Pointer(&raw[0]))
		x11.XNextEvent(h.dpy, unsafe.Pointer(ev))
		h.one(ev)
	}
}

type xEvent struct {
	kind    int32
	_       int32
	serial  uint64
	send    int32
	_       int32
	display uintptr
	window  uint64
	root    uint64
	sub     uint64
	time    uint64
	x, y    int32
	xroot   int32
	yroot   int32
	state   uint32
	button  uint32
	same    int32
}

type xConfigure struct {
	kind    int32
	_       int32
	serial  uint64
	send    int32
	_       int32
	display uintptr
	event   uint64
	window  uint64
	x, y    int32
	width   int32
	height  int32
}

func (h *xlibHost) one(ev *xEvent) {
	if h == nil || ev == nil || h.screen == nil {
		return
	}
	switch ev.kind {
	case 2, 3:
		h.screen.emit(Input{Kind: InputKey, X: int(ev.x), Y: int(ev.y), Code: ev.button, Pressed: ev.kind == 2})
	case 4:
		h.screen.emit(Input{Kind: InputPointer, X: int(ev.x), Y: int(ev.y), Button: int(ev.button), Pressed: true})
	case 5:
		h.screen.emit(Input{Kind: InputPointer, X: int(ev.x), Y: int(ev.y), Button: int(ev.button)})
	case 6:
		h.screen.emit(Input{Kind: InputPointer, X: int(ev.x), Y: int(ev.y)})
	case 22:
		cfg := (*xConfigure)(unsafe.Pointer(&ev))
		if cfg.width > 0 && cfg.height > 0 {
			h.screen.emit(Input{Kind: InputResize, X: int(cfg.width), Y: int(cfg.height)})
		}
	case 17:
		h.screen.emit(Input{Kind: InputClose})
	case 12:
		h.screen.emit(Input{Kind: InputExpose})
	}
}

func (h *xlibHost) create(d *Device, w *wsi) (uint64, error) {
	if err := d.api.bind(d.api.getInstanceProcAddr, d.inst, "vkCreateXlibSurfaceKHR", &h.createXlib); err != nil {
		return 0, err
	}
	_ = w
	info := xlibSurfaceInfo{sType: 1000004000, dpy: h.dpy, window: h.win}
	var surface uint64
	if err := check(h.createXlib(d.inst, &info, 0, &surface)); err != nil {
		return 0, fmt.Errorf("%w: %w", ErrXlibHostCreateXlibSurface, err)
	}
	return surface, nil
}

func (h *xlibHost) destroy() {
	if h == nil || h.borrowed || h.dpy == 0 {
		return
	}
	x11, err := loadX11()
	if err == nil {
		if h.win != 0 {
			x11.XDestroyWindow(h.dpy, h.win)
		}
		x11.XCloseDisplay(h.dpy)
	}
	h.dpy, h.win = 0, 0
}

var (
	bindX11    = native.BindOnce[X11](0, "libX11.so.6")
	bindX11Opt = native.BindOnce[X11Opt](0, "libX11.so.6")
)

// loadX11 binds libX11, including the optional symbols, and returns the hard set.
func loadX11() (X11, error) {
	x11, err := bindX11()
	if err != nil {
		return X11{}, fmt.Errorf("%w: libX11: %v", ErrUnavailable, err)
	}
	if _, err = bindX11Opt(); err != nil {
		return X11{}, fmt.Errorf("%w: libX11: %v", ErrUnavailable, err)
	}
	return x11, nil
}

const xInputMask int64 = 1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<6 | 1<<15 | 1<<17

func xOpen() (uintptr, error) {
	x11, err := loadX11()
	if err != nil {
		return 0, err
	}
	dpy := x11.XOpenDisplay(nil)
	if dpy == 0 {
		return 0, fmt.Errorf("%w: DISPLAY", ErrUnavailable)
	}
	return dpy, nil
}
