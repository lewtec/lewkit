//go:build windows

package win32

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/release"
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	wsExAcceptFiles    = 0x00000010
	wsExAppWindow      = 0x00040000
	colorWindow        = 5
	swpNoSize          = 0x0001
	swpNoMove          = 0x0002
	swpShowWindow      = 0x0040
	wmDropFiles        = 0x0233
	wmSetIcon          = 0x0080
	iconBig            = 1
	iconSmall          = 0
	imageIcon          = 1
	lrShared           = 0x8000
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
	procRegisterClassExW         = native.ProcOf("user32.dll", "RegisterClassExW")
	procCreateWindowExW          = native.ProcOf("user32.dll", "CreateWindowExW")
	procDefWindowProcW           = native.ProcOf("user32.dll", "DefWindowProcW")
	procGetMessageW              = native.ProcOf("user32.dll", "GetMessageW")
	procTranslateMessage         = native.ProcOf("user32.dll", "TranslateMessage")
	procDispatchMessageW         = native.ProcOf("user32.dll", "DispatchMessageW")
	procShowWindow               = native.ProcOf("user32.dll", "ShowWindow")
	procGetForegroundWindow      = native.ProcOf("user32.dll", "GetForegroundWindow")
	procGetWindowThreadProcessId = native.ProcOf("user32.dll", "GetWindowThreadProcessId")
	procSetForegroundWindow      = native.ProcOf("user32.dll", "SetForegroundWindow")
	procBringWindowToTop         = native.ProcOf("user32.dll", "BringWindowToTop")
	procAttachThreadInput        = native.ProcOf("user32.dll", "AttachThreadInput")
	procAllowSetForegroundWindow = native.ProcOf("user32.dll", "AllowSetForegroundWindow")
	procGetCurrentProcessId      = native.ProcOf("kernel32.dll", "GetCurrentProcessId")
	procGetCurrentThreadId       = native.ProcOf("kernel32.dll", "GetCurrentThreadId")
	procGetDC                    = native.ProcOf("user32.dll", "GetDC")
	procReleaseDC                = native.ProcOf("user32.dll", "ReleaseDC")
	procBeginPaint               = native.ProcOf("user32.dll", "BeginPaint")
	procEndPaint                 = native.ProcOf("user32.dll", "EndPaint")
	procDestroyWindow            = native.ProcOf("user32.dll", "DestroyWindow")
	procSetWindowPos             = native.ProcOf("user32.dll", "SetWindowPos")
	procGetClientRect            = native.ProcOf("user32.dll", "GetClientRect")
	procPostQuitMessage          = native.ProcOf("user32.dll", "PostQuitMessage")
	procGetModuleHandleW         = native.ProcOf("kernel32.dll", "GetModuleHandleW")
	procStretchDIBits            = native.ProcOf("gdi32.dll", "StretchDIBits")
	procGetDeviceCaps            = native.ProcOf("gdi32.dll", "GetDeviceCaps")
	procDragAcceptFiles          = native.ProcOf("shell32.dll", "DragAcceptFiles")
	procDragQueryFileW           = native.ProcOf("shell32.dll", "DragQueryFileW")
	procDragFinish               = native.ProcOf("shell32.dll", "DragFinish")
	procLoadImageW               = native.ProcOf("user32.dll", "LoadImageW")
	procSendMessageW             = native.ProcOf("user32.dll", "SendMessageW")
	classOnce                    sync.Once
	classAtom                    uintptr
	classErr                     error
	className                    = syscall.StringToUTF16Ptr(release.Name() + ".driver.window")
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

// paintStruct matches PAINTSTRUCT. On 64-bit Windows the fields occupy 68 bytes.
type paintStruct struct {
	hdc       uintptr
	erase     int32
	rcLeft    int32
	rcTop     int32
	rcRight   int32
	rcBottom  int32
	restore   int32
	incUpdate int32
	reserved  [32]byte
}

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
	if ctx == nil {
		return nil, fmt.Errorf("window: nil context")
	}
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
	out := &win{Buffer: buf, title: cfg.Title, icon: cfg.Icon, cw: w, ch: h, want: window.WantSize{Width: w, Height: h}}
	go func() {
		runtime.LockOSThread()
		ready <- out.create()
		if out.hwnd != 0 {
			out.pump()
		}
	}()
	select {
	case <-ctx.Done():
		go func() {
			if err := <-ready; err == nil {
				_ = out.Close()
			}
		}()
		return nil, ctx.Err()
	case err := <-ready:
		if err != nil {
			return nil, err
		}
	}
	window.CloseWhenDone(ctx, out)
	return out, nil
}

type win struct {
	*window.Buffer
	mu     sync.Mutex
	hwnd   uintptr
	title  string
	icon   image.Image
	cw, ch int
	want   window.WantSize
	// gdi is set after a CPU Draw has published a front buffer.
	// A Vulkan window never Draws, so WM_PAINT leaves the swapchain visible.
	gdi atomic.Bool
}

func (w *win) Surface() window.Surface {
	if w == nil || w.hwnd == 0 {
		return window.Surface{}
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	return window.Surface{Kind: window.SurfaceWin32, A: w.hwnd, B: inst}
}

func (w *win) Size() image.Point {
	return window.HostSize(w.Buffer, &w.mu, &w.want)
}

func (w *win) Frame() *image.RGBA {
	return window.HostFrame(w.Buffer, w.Size())
}

func (w *win) create() error {
	classOnce.Do(func() {
		inst, _, _ := procGetModuleHandleW.Call(0)
		wc := wndClassEx{
			size:       uint32(unsafe.Sizeof(wndClassEx{})),
			wndProc:    syscall.NewCallback(wndProc),
			instance:   inst,
			background: colorWindow + 1,
			className:  className,
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
	owner := foregroundProcessWindow()
	hwnd, _, e := procCreateWindowExW.Call(
		wsExAcceptFiles|wsExAppWindow, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow|wsVisible,
		uintptr(cwUseDefault), uintptr(cwUseDefault), uintptr(w.cw+16), uintptr(w.ch+39),
		owner, 0, inst, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("%w: %v", window.ErrInit, e)
	}
	w.hwnd = hwnd
	windows.Store(hwnd, w)
	window.RegisterUI(hwnd)
	procDragAcceptFiles.Call(hwnd, 1)
	if w.icon != nil {
		window.ApplyWindowIcon(hwnd, w.icon)
	} else if h := moduleIcon(inst); h != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconBig, h)
		procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, h)
	}
	procShowWindow.Call(hwnd, swShow)
	raiseWindow(hwnd)
	return nil
}

func foregroundProcessWindow() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	current, _, _ := procGetCurrentProcessId.Call()
	if uint32(current) != pid {
		return 0
	}
	return hwnd
}

func raiseWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	procAllowSetForegroundWindow.Call(uintptr(0xFFFFFFFF))
	fg, _, _ := procGetForegroundWindow.Call()
	var fgThread uintptr
	if fg != 0 {
		fgThread, _, _ = procGetWindowThreadProcessId.Call(fg, 0)
	}
	our, _, _ := procGetCurrentThreadId.Call()
	attached := false
	if fgThread != 0 && fgThread != our {
		r, _, _ := procAttachThreadInput.Call(our, fgThread, 1)
		attached = r != 0
	}
	procSetForegroundWindow.Call(hwnd)
	procBringWindowToTop.Call(hwnd)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	if attached {
		procAttachThreadInput.Call(our, fgThread, 0)
	}
}

func moduleIcon(inst uintptr) uintptr {
	h, _, _ := procLoadImageW.Call(inst, 1, imageIcon, 0, 0, lrShared)
	return h
}

func dropPaths(hdrop uintptr) []string {
	count, _, _ := procDragQueryFileW.Call(hdrop, 0xFFFFFFFF, 0, 0)
	paths := make([]string, 0, count)
	for i := uintptr(0); i < count; i++ {
		buf := make([]uint16, 32768)
		n, _, _ := procDragQueryFileW.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			continue
		}
		paths = append(paths, syscall.UTF16ToString(buf))
	}
	return paths
}

func (w *win) pump() {
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		// A file dialog has to start outside the window procedure.
		if window.DeliverUI(uintptr(m.message), m.wParam) {
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (w *win) Draw() error {
	err := window.SwapBlit(w.Buffer, w.blit)
	if err == nil {
		w.gdi.Store(true)
	}
	return err
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
		window.ForgetUI(hwnd)
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
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return fmt.Errorf("%w: dc", window.ErrPresent)
	}
	defer procReleaseDC.Call(hwnd, hdc)
	return w.stretch(hdc)
}

func (w *win) stretch(hdc uintptr) error {
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
	if width == 0 || height == 0 || len(bgra) == 0 {
		return nil
	}
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
	runtime.KeepAlive(bgra)
	runtime.KeepAlive(&bi)
	if r == 0 {
		return fmt.Errorf("%w: %v", window.ErrPresent, err)
	}
	return nil
}

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if window.DeliverUI(msg, wparam) {
		return 1
	}
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
		var painted paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&painted)))
		if hdc != 0 && w.gdi.Load() {
			_ = w.stretch(hdc)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&painted)))
		w.Emit(window.Expose{})
		return 0
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
	case wmDropFiles:
		w.Emit(window.Drop{Paths: dropPaths(wparam)})
		procDragFinish.Call(wparam)
		return 0
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
