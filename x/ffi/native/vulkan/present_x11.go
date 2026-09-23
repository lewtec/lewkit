//go:build linux

package vulkan

import (
	"fmt"

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
	dpy        uintptr
	win        uint64
	createXlib func(inst uintptr, info *xlibSurfaceInfo, alloc uintptr, surface *uint64) int32
}

func loadHost(w *wsi, d *Device) error {
	return nil
}

func openHost(width, height int, title string) (hostSurface, error) {
	dpy, err := xOpen()
	if err != nil {
		return nil, err
	}
	screen := xDefaultScreen(dpy)
	root := xRootWindow(dpy, screen)
	win := xCreateSimple(dpy, root, 0, 0, uint32(width), uint32(height), 0, xBlackPixel(dpy, screen), xWhitePixel(dpy, screen))
	if win == 0 {
		xCloseDisplay(dpy)
		return nil, fmt.Errorf("%w: x window", ErrUnavailable)
	}
	name := cstr(title)
	xStoreName(dpy, win, name)
	xSelectInput(dpy, win, 1<<15)
	xMapWindow(dpy, win)
	xFlush(dpy)
	return &xlibHost{dpy: dpy, win: win}, nil
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
	if h == nil || h.dpy == 0 {
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
	xlibOnce       uint32
)

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
