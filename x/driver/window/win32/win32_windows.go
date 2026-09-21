//go:build windows

package win32

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/window"
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	cwUseDefault       = 0x80000000
	swShow             = 5
	wmDestroy          = 0x0002
	wmSize             = 0x0005
	wmPaint            = 0x000F
	wmClose            = 0x0010
	wmKeyDown          = 0x0100
	wmKeyUp            = 0x0101
	wmMouseMove        = 0x0200
	wmLButtonDown      = 0x0201
	wmLButtonUp        = 0x0202
	wmRButtonDown      = 0x0204
	wmRButtonUp        = 0x0205
	wmMButtonDown      = 0x0207
	wmMButtonUp        = 0x0208
	wmMouseWheel       = 0x020A
	mkLButton          = 0x0001
	mkRButton          = 0x0002
	mkMButton          = 0x0010
	biRGB              = 0
	dibRGBColors       = 0
	srcCopy            = 0x00CC0020
	sizeRestored       = 0
	sizeMaximized      = 2
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procStretchDIBits    = gdi32.NewProc("StretchDIBits")
	procGetDeviceCaps    = gdi32.NewProc("GetDeviceCaps")
	classOnce            sync.Once
	classAtom            uintptr
	classErr             error
	className            = syscall.StringToUTF16Ptr("lewkit.driver.window")
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type rect struct {
	left, top, right, bottom int32
}

type point struct{ x, y int32 }

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type bitmapInfo struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

var windows sync.Map // hwnd -> *win

const vrefresh = 116

func desktopFramePeriod() time.Duration {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return 0
	}
	hz, _, _ := procGetDeviceCaps.Call(hdc, vrefresh)
	procReleaseDC.Call(0, hdc)
	if hz <= 1 {
		return 0
	}
	return time.Second / time.Duration(hz)
}

func (wdriver) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	ready := make(chan error, 1)
	buf := window.NewBuffer(w, h)
	period := cfg.Period
	if period <= 0 {
		period = desktopFramePeriod()
	}
	buf.SetFramePeriod(period)
	out := &win{Buffer: buf, title: cfg.Title, cw: w, ch: h, want: window.WantSize{Width: w, Height: h}}
	go func() {
		runtime.LockOSThread()
		ready <- out.create()
		if out.hwnd != 0 {
			out.pump()
		}
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	window.CloseWhenDone(ctx, out)
	return out, nil
}

type win struct {
	*window.Buffer
	mu     sync.Mutex
	hwnd   uintptr
	title  string
	cw, ch int
	want   window.WantSize
}

func (w *win) Size() image.Point {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.want.Point(w.Buffer.Size())
}

func (w *win) Frame() *image.RGBA {
	want := w.Size()
	if want.X > 0 && want.Y > 0 {
		_, _ = w.Buffer.EnsureSize(want)
	}
	return w.Buffer.Frame()
}

func (w *win) create() error {
	classOnce.Do(func() {
		inst, _, _ := procGetModuleHandleW.Call(0)
		wc := wndClassEx{
			size:      uint32(unsafe.Sizeof(wndClassEx{})),
			wndProc:   syscall.NewCallback(wndProc),
			instance:  inst,
			className: className,
		}
		classAtom, _, classErr = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if classAtom == 0 {
			classErr = fmt.Errorf("%w: %v", window.ErrInit, classErr)
		} else {
			classErr = nil
		}
	})
	if classErr != nil {
		return classErr
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	title, err := syscall.UTF16PtrFromString(w.title)
	if err != nil {
		return err
	}
	hwnd, _, e := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow|wsVisible,
		uintptr(cwUseDefault), uintptr(cwUseDefault), uintptr(w.cw+16), uintptr(w.ch+39),
		0, 0, inst, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("%w: %v", window.ErrInit, e)
	}
	w.hwnd = hwnd
	windows.Store(hwnd, w)
	procShowWindow.Call(hwnd, swShow)
	return nil
}

func (w *win) pump() {
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (w *win) Draw() error {
	return window.SwapBlit(w.Buffer, w.blit)
}

func (w *win) Resize(size image.Point) error {
	w.mu.Lock()
	w.want.Width, w.want.Height = size.X, size.Y
	w.mu.Unlock()
	if err := w.Buffer.Resize(size); err != nil {
		return err
	}
	w.mu.Lock()
	hwnd := w.hwnd
	w.mu.Unlock()
	if hwnd == 0 {
		return window.ErrClosed
	}
	const swpNoMove = 0x0002
	const swpNoZOrder = 0x0004
	procSetWindowPos.Call(hwnd, 0, 0, 0, uintptr(size.X+16), uintptr(size.Y+39), swpNoMove|swpNoZOrder)
	return nil
}

func (w *win) Close() error {
	w.mu.Lock()
	hwnd := w.hwnd
	w.hwnd = 0
	w.mu.Unlock()
	_ = w.Buffer.Close()
	if hwnd != 0 {
		windows.Delete(hwnd)
		procDestroyWindow.Call(hwnd)
	}
	return nil
}

func (w *win) blit() error {
	w.mu.Lock()
	hwnd := w.hwnd
	w.mu.Unlock()
	if hwnd == 0 {
		return window.ErrClosed
	}
	var width, height int
	var bgra []byte
	w.WithFront(func(src *image.RGBA) {
		width, height = src.Rect.Dx(), src.Rect.Dy()
		if width == 0 || height == 0 {
			return
		}
		bgra = make([]byte, len(src.Pix))
		window.ToBGRA(bgra, src)
	})
	if width == 0 || height == 0 {
		return nil
	}
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return fmt.Errorf("%w: dc", window.ErrPresent)
	}
	defer procReleaseDC.Call(hwnd, hdc)
	bi := bitmapInfo{
		size:        uint32(unsafe.Sizeof(bitmapInfo{})),
		width:       int32(width),
		height:      -int32(height),
		planes:      1,
		bitCount:    32,
		compression: biRGB,
	}
	r, _, err := procStretchDIBits.Call(
		hdc,
		0, 0, uintptr(width), uintptr(height),
		0, 0, uintptr(width), uintptr(height),
		uintptr(unsafe.Pointer(&bgra[0])),
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
		srcCopy,
	)
	if r == 0 {
		return fmt.Errorf("%w: %v", window.ErrPresent, err)
	}
	return nil
}

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	v, ok := windows.Load(hwnd)
	if !ok {
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	}
	w := v.(*win)
	switch msg {
	case wmSize:
		if wparam == sizeRestored || wparam == sizeMaximized {
			width := int(lparam & 0xFFFF)
			height := int((lparam >> 16) & 0xFFFF)
			if width > 0 && height > 0 {
				window.SetWant(w.Buffer, &w.mu, &w.want, width, height)
			}
		}
	case wmPaint:
		w.Emit(window.Expose{})
		_ = w.blit()
	case wmMouseMove:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Buttons: win32Buttons(wparam)})
	case wmLButtonDown:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 1, Pressed: true, Buttons: win32Buttons(wparam) | window.ButtonLeft})
	case wmLButtonUp:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 1, Pressed: false, Buttons: win32Buttons(wparam) &^ window.ButtonLeft})
	case wmRButtonDown:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 2, Pressed: true, Buttons: win32Buttons(wparam) | window.ButtonRight})
	case wmRButtonUp:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 2, Pressed: false, Buttons: win32Buttons(wparam) &^ window.ButtonRight})
	case wmMButtonDown:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 3, Pressed: true, Buttons: win32Buttons(wparam) | window.ButtonMiddle})
	case wmMButtonUp:
		w.Emit(window.Pointer{Pos: win32Pos(lparam), Button: 3, Pressed: false, Buttons: win32Buttons(wparam) &^ window.ButtonMiddle})
	case wmMouseWheel:
		delta := int16(wparam >> 16)
		w.Emit(window.Scroll{Pos: win32Pos(lparam), Delta: image.Pt(0, -int(delta)/12)})
	case wmKeyDown:
		w.Emit(window.Key{Code: uint32(wparam), Pressed: true, Repeat: lparam&0x40000000 != 0, Mod: win32KeyMod(wparam)})
	case wmKeyUp:
		w.Emit(window.Key{Code: uint32(wparam), Pressed: false, Mod: win32KeyMod(wparam)})
	case wmClose:
		_ = w.Close()
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

func win32Pos(lparam uintptr) image.Point {
	return image.Pt(int(int16(lparam)), int(int16(lparam>>16)))
}

func win32Buttons(wparam uintptr) int {
	var b int
	if wparam&mkLButton != 0 {
		b |= window.ButtonLeft
	}
	if wparam&mkRButton != 0 {
		b |= window.ButtonRight
	}
	if wparam&mkMButton != 0 {
		b |= window.ButtonMiddle
	}
	return b
}

func win32KeyMod(wparam uintptr) window.Modifier {
	var m window.Modifier
	if wparam == 0x10 {
		m |= window.ModShift
	}
	if wparam == 0x11 {
		m |= window.ModCtrl
	}
	if wparam == 0x12 {
		m |= window.ModAlt
	}
	if wparam == 0x5B || wparam == 0x5C {
		m |= window.ModSuper
	}
	return m
}
