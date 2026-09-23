//go:build windows

package vulkan

import (
	"fmt"
	"unicode/utf16"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const extHostSurface = "VK_KHR_win32_surface"

func surfaceExtensions() []string { return []string{extHostSurface} }

func loadHost(w *wsi, d *Device) error { return nil }

type win32SurfaceInfo struct {
	sType     int32
	pNext     uintptr
	flags     uint32
	hinstance uintptr
	hwnd      uintptr
}

type win32Host struct {
	hwnd       uintptr
	instance   uintptr
	createSurf func(inst uintptr, info *win32SurfaceInfo, alloc uintptr, surface *uint64) int32
}

func openHost(width, height int, title string) (hostSurface, error) {
	if err := loadUser32(); err != nil {
		return nil, err
	}
	inst := getModuleHandle(nil)
	class, err := utf16z("lewkitvulkan")
	if err != nil {
		return nil, err
	}
	if !classReady {
		proc, err := native.Symbol(user32, "DefWindowProcW")
		if err != nil {
			return nil, fmt.Errorf("%w: DefWindowProcW", ErrUnavailable)
		}
		wc := wndClassW{
			wndProc:   proc,
			instance:  inst,
			className: class,
		}
		if registerClass(&wc) == 0 {
			return nil, fmt.Errorf("%w: register class", ErrUnavailable)
		}
		classReady = true
	}
	name, err := utf16z(title)
	if err != nil {
		return nil, err
	}
	hwnd := createWindow(0, class, name, wsOverlappedWindow, cwUseDefault, cwUseDefault, int32(width), int32(height), 0, 0, inst, 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("%w: win32 window", ErrUnavailable)
	}
	showWindow(hwnd, swShow)
	return &win32Host{hwnd: hwnd, instance: inst}, nil
}

func (h *win32Host) create(d *Device, w *wsi) (uint64, error) {
	_ = w
	if err := d.api.bind(d.api.getInstanceProcAddr, d.inst, "vkCreateWin32SurfaceKHR", &h.createSurf); err != nil {
		return 0, err
	}
	info := win32SurfaceInfo{sType: 1000009000, hinstance: h.instance, hwnd: h.hwnd}
	var surface uint64
	if err := check(h.createSurf(d.inst, &info, 0, &surface)); err != nil {
		return 0, fmt.Errorf("win32 surface: %w", err)
	}
	return surface, nil
}

func (h *win32Host) destroy() {
	if h == nil || h.hwnd == 0 {
		return
	}
	destroyWindow(h.hwnd)
	h.hwnd = 0
}

const (
	wsOverlappedWindow = 0x00CF0000
	cwUseDefault       = int32(-2147483648)
	swShow             = 5
)

type wndClassW struct {
	style     uint32
	wndProc   uintptr
	clsExtra  int32
	wndExtra  int32
	instance  uintptr
	icon      uintptr
	cursor    uintptr
	brush     uintptr
	menuName  *uint16
	className *uint16
}

var (
	user32          uintptr
	classReady      bool
	getModuleHandle func(name *uint16) uintptr
	registerClass   func(wc *wndClassW) uint16
	createWindow    func(ex uint32, class, title *uint16, style uint32, x, y, w, h int32, parent, menu, inst, param uintptr) uintptr
	showWindow      func(hwnd uintptr, cmd int32) int32
	destroyWindow   func(hwnd uintptr) int32
)

func loadUser32() error {
	if user32 != 0 {
		return nil
	}
	lib, err := native.Open("user32.dll", 0)
	if err != nil {
		return fmt.Errorf("%w: user32", ErrUnavailable)
	}
	k32, err := native.Open("kernel32.dll", 0)
	if err != nil {
		return fmt.Errorf("%w: kernel32", ErrUnavailable)
	}
	native.Func(k32, "GetModuleHandleW", &getModuleHandle)
	native.Func(lib, "RegisterClassW", &registerClass)
	native.Func(lib, "CreateWindowExW", &createWindow)
	native.Func(lib, "ShowWindow", &showWindow)
	native.Func(lib, "DestroyWindow", &destroyWindow)
	user32 = lib
	if getModuleHandle == nil || registerClass == nil || createWindow == nil {
		return fmt.Errorf("%w: win32 window procs", ErrUnavailable)
	}
	return nil
}

func utf16z(s string) (*uint16, error) {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	return &u[0], nil
}
