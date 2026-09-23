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
	screen     *Screen
	hwnd       uintptr
	instance   uintptr
	createSurf func(inst uintptr, info *win32SurfaceInfo, alloc uintptr, surface *uint64) int32
}

type win32Msg struct {
	hwnd    uintptr
	message uint32
	_       uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	_       uint32
	x, y    int32
}

func (h *win32Host) one(msg win32Msg) {
	if h.screen == nil {
		return
	}
	xy := func() (int, int) {
		v := int32(msg.lparam)
		return int(int16(v)), int(int16(v >> 16))
	}
	switch msg.message {
	case 0x0005:
		h.screen.emit(Input{Kind: InputResize, X: int(int16(msg.lparam)), Y: int(int16(msg.lparam >> 16))})
	case 0x0010, 0x0002:
		h.screen.emit(Input{Kind: InputClose})
	case 0x0200:
		x, y := xy()
		h.screen.emit(Input{Kind: InputPointer, X: x, Y: y})
	case 0x0201, 0x0204, 0x0207:
		x, y := xy()
		h.screen.emit(Input{Kind: InputPointer, X: x, Y: y, Button: winButton(msg.message), Pressed: true})
	case 0x0202, 0x0205, 0x0208:
		x, y := xy()
		h.screen.emit(Input{Kind: InputPointer, X: x, Y: y, Button: winButton(msg.message)})
	case 0x020A:
		x, y := xy()
		h.screen.emit(Input{Kind: InputScroll, X: x, Y: y, DY: int(int16(msg.wparam>>16)) / 120})
	case 0x0100, 0x0101:
		h.screen.emit(Input{Kind: InputKey, Code: uint32(msg.wparam), Pressed: msg.message == 0x0100, Repeat: msg.lparam&(1<<30) != 0})
	case 0x000F:
		h.screen.emit(Input{Kind: InputExpose})
	}
}

func winButton(msg uint32) int {
	switch msg {
	case 0x0204, 0x0205:
		return 2
	case 0x0207, 0x0208:
		return 3
	default:
		return 1
	}
}

func openHost(screen *Screen, width, height int, title string) (hostSurface, error) {
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
	return &win32Host{screen: screen, hwnd: hwnd, instance: inst}, nil
}

func (h *win32Host) poll() {
	if h == nil || h.hwnd == 0 || peekMessage == nil {
		return
	}
	var msg win32Msg
	for peekMessage(&msg, h.hwnd, 0, 0, 1) {
		h.one(msg)
		if translateMessage != nil {
			translateMessage(&msg)
		}
		if dispatchMessage != nil {
			dispatchMessage(&msg)
		}
	}
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
	user32           uintptr
	classReady       bool
	getModuleHandle  func(name *uint16) uintptr
	registerClass    func(wc *wndClassW) uint16
	createWindow     func(ex uint32, class, title *uint16, style uint32, x, y, w, h int32, parent, menu, inst, param uintptr) uintptr
	showWindow       func(hwnd uintptr, cmd int32) int32
	destroyWindow    func(hwnd uintptr) int32
	peekMessage      func(msg *win32Msg, hwnd uintptr, min, max, remove uint32) bool
	translateMessage func(msg *win32Msg) bool
	dispatchMessage  func(msg *win32Msg) uintptr
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
	native.Func(lib, "PeekMessageW", &peekMessage)
	native.Func(lib, "TranslateMessage", &translateMessage)
	native.Func(lib, "DispatchMessageW", &dispatchMessage)
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
