//go:build windows

package vulkan

import (
	"fmt"
	"unicode/utf16"
	"unsafe"

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
	borrowed   bool
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

func attachHost(s *Screen, kind int, a, b uintptr) (hostSurface, error) {
	if kind != 2 || a == 0 {
		return nil, ErrUnavailable
	}
	return &win32Host{screen: s, hwnd: a, instance: b, borrowed: true}, nil
}

func (h *win32Host) poll() {
	if h == nil || h.hwnd == 0 {
		return
	}
	user, err := loadWin32()
	if err != nil {
		return
	}
	var msg win32Msg
	for user.PeekMessageW(unsafe.Pointer(&msg), h.hwnd, 0, 0, 1) {
		h.one(msg)
		user.TranslateMessage(unsafe.Pointer(&msg))
		user.DispatchMessageW(unsafe.Pointer(&msg))
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
		return 0, fmt.Errorf("%w: %w", ErrWin32HostCreateWin32Surface, err)
	}
	return surface, nil
}

func (h *win32Host) destroy() {
	if h == nil || h.borrowed || h.hwnd == 0 {
		return
	}
	user, err := loadWin32()
	if err == nil {
		user.DestroyWindow(h.hwnd)
	}
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
	bindUser32   = native.BindOnce[User32](0, "user32.dll")
	bindKernel32 = native.BindOnce[Kernel32](0, "kernel32.dll")
)

// loadWin32 binds user32 and kernel32 and returns the user32 symbols from that load.
func loadWin32() (User32, error) {
	user, err := bindUser32()
	if err != nil {
		return User32{}, fmt.Errorf("%w: user32: %v", ErrUnavailable, err)
	}
	if _, err = bindKernel32(); err != nil {
		return User32{}, fmt.Errorf("%w: kernel32: %v", ErrUnavailable, err)
	}
	return user, nil
}

func utf16z(s string) (*uint16, error) {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	return &u[0], nil
}
