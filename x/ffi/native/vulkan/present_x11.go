//go:build linux

package vulkan

import (
	"fmt"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
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

func openHost(screen *Screen, width, height int, title string) (hostSurface, error) {
	dpy, err := xOpen()
	if err != nil {
		return nil, err
	}
	display := xDefaultScreen(dpy)
	root := xRootWindow(dpy, display)
	win := xCreateSimple(dpy, root, 0, 0, uint32(width), uint32(height), 0, xBlackPixel(dpy, display), xWhitePixel(dpy, display))
	if win == 0 {
		xCloseDisplay(dpy)
		return nil, fmt.Errorf("%w: x window", ErrUnavailable)
	}
	name := cstr(title)
	xStoreName(dpy, win, name)
	xSelectInput(dpy, win, xInputMask)
	_ = screen
	xMapWindow(dpy, win)
	xFlush(dpy)
	return &xlibHost{screen: screen, dpy: dpy, win: win}, nil
}

func (h *xlibHost) poll() {
	if h == nil || h.dpy == 0 || xPending == nil {
		return
	}
	for xPending(h.dpy) > 0 {
		var raw [192]byte
		ev := (*xEvent)(unsafe.Pointer(&raw[0]))
		xNext(h.dpy, ev)
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
		return 0, fmt.Errorf("xlib surface: %w", err)
	}
	return surface, nil
}

func (h *xlibHost) destroy() {
	if h == nil || h.borrowed || h.dpy == 0 {
		return
	}
	if h.win != 0 {
		xDestroyWindow(h.dpy, h.win)
	}
	xCloseDisplay(h.dpy)
	h.dpy, h.win = 0, 0
}

var (
	xOpenDisplay   func(name *byte) uintptr
	xCloseDisplay  func(dpy uintptr) int32
	xDefaultScreen func(dpy uintptr) int32
	xRootWindow    func(dpy uintptr, screen int32) uint64
	xBlackPixel    func(dpy uintptr, screen int32) uint64
	xWhitePixel    func(dpy uintptr, screen int32) uint64
	xCreateSimple  func(dpy uintptr, parent uint64, x, y int32, w, h, border uint32, borderC, bg uint64) uint64
	xStoreName     func(dpy uintptr, win uint64, name *byte) int32
	xMapWindow     func(dpy uintptr, win uint64) int32
	xFlush         func(dpy uintptr) int32
	xDestroyWindow func(dpy uintptr, win uint64) int32
	xSelectInput   func(dpy uintptr, win uint64, mask int64) int32
	xPending       func(dpy uintptr) int32
	xNext          func(dpy uintptr, ev *xEvent) int32
	xlibOnce       uint32
)

const xInputMask int64 = 1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<6 | 1<<15 | 1<<17

func xOpen() (uintptr, error) {
	if xlibOnce == 0 {
		lib, err := native.Open("libX11.so.6", 0)
		if err != nil {
			return 0, fmt.Errorf("%w: libX11", ErrUnavailable)
		}
		native.Func(lib, "XOpenDisplay", &xOpenDisplay)
		native.Func(lib, "XCloseDisplay", &xCloseDisplay)
		native.Func(lib, "XDefaultScreen", &xDefaultScreen)
		native.Func(lib, "XRootWindow", &xRootWindow)
		native.Func(lib, "XBlackPixel", &xBlackPixel)
		native.Func(lib, "XWhitePixel", &xWhitePixel)
		native.Func(lib, "XCreateSimpleWindow", &xCreateSimple)
		native.Func(lib, "XStoreName", &xStoreName)
		native.Func(lib, "XMapWindow", &xMapWindow)
		native.Func(lib, "XFlush", &xFlush)
		native.Func(lib, "XDestroyWindow", &xDestroyWindow)
		native.Func(lib, "XSelectInput", &xSelectInput)
		native.Func(lib, "XPending", &xPending)
		native.Func(lib, "XNextEvent", &xNext)
		xlibOnce = 1
	}
	if xOpenDisplay == nil {
		return 0, fmt.Errorf("%w: XOpenDisplay", ErrUnavailable)
	}
	dpy := xOpenDisplay(nil)
	if dpy == 0 {
		return 0, fmt.Errorf("%w: DISPLAY", ErrUnavailable)
	}
	return dpy, nil
}
