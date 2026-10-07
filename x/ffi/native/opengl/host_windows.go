//go:build windows

package opengl

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// kindWin32 is window.SurfaceWin32: HWND in a, HINSTANCE in b.
const kindWin32 = 2

const (
	wsPopup = 0x80000000
	csOwnDC = 0x0020
	// ERROR_CLASS_ALREADY_EXISTS
	errClassExists = syscall.Errno(1410)

	wglContextMajor = 0x2091
	wglContextMinor = 0x2092
	wglContextMask  = 0x9126
	wglCoreProfile  = 0x00000001
)

var (
	procRegisterClassW    = native.ProcOf("user32.dll", "RegisterClassW")
	procCreateWindowExW   = native.ProcOf("user32.dll", "CreateWindowExW")
	procDestroyWindow     = native.ProcOf("user32.dll", "DestroyWindow")
	procDefWindowProcW    = native.ProcOf("user32.dll", "DefWindowProcW")
	procGetDC             = native.ProcOf("user32.dll", "GetDC")
	procReleaseDC         = native.ProcOf("user32.dll", "ReleaseDC")
	procGetModuleHandleW  = native.ProcOf("kernel32.dll", "GetModuleHandleW")
	procChoosePixelFormat = native.ProcOf("gdi32.dll", "ChoosePixelFormat")
	procSetPixelFormat    = native.ProcOf("gdi32.dll", "SetPixelFormat")
	procGetPixelFormat    = native.ProcOf("gdi32.dll", "GetPixelFormat")
	procSwapBuffers       = native.ProcOf("gdi32.dll", "SwapBuffers")
	procWglCreateContext  = native.ProcOf("opengl32.dll", "wglCreateContext")
	procWglDeleteContext  = native.ProcOf("opengl32.dll", "wglDeleteContext")
	procWglMakeCurrent    = native.ProcOf("opengl32.dll", "wglMakeCurrent")
	procWglGetProcAddress = native.ProcOf("opengl32.dll", "wglGetProcAddress")
)

// wndClassW matches WNDCLASSW. Go aligns the pointer fields the same way
// the Windows ABI does on 386 and amd64.
type wndClassW struct {
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menu       uintptr
	class      uintptr
}

type wglContext struct {
	hwnd, hdc, ctx uintptr
	own            bool
	glLib          uintptr
	getProc        func(uintptr) uintptr
}

var (
	classOnce sync.Once
	classErr  error
	className *uint16
	classProc uintptr
)

func openglWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

func ensureClass() error {
	classOnce.Do(func() {
		name, err := syscall.UTF16PtrFromString("lewkit-opengl")
		if err != nil {
			classErr = err
			return
		}
		className = name
		classProc = syscall.NewCallback(openglWndProc)
		inst, _, _ := procGetModuleHandleW.Call(0)
		wc := wndClassW{
			style:    csOwnDC,
			wndProc:  classProc,
			instance: inst,
			class:    uintptr(unsafe.Pointer(className)),
		}
		atom, _, callErr := procRegisterClassW.Call(uintptr(unsafe.Pointer(&wc)))
		runtime.KeepAlive(&wc)
		if atom == 0 && !errors.Is(callErr, errClassExists) {
			classErr = fmt.Errorf("%w: RegisterClassW %v", ErrUnavailable, callErr)
		}
	})
	return classErr
}

func pixelFormat() []byte {
	pfd := make([]byte, 40)
	pfd[0] = 40 // nSize
	pfd[2] = 1  // nVersion
	pfd[4] = 0x25
	pfd[9] = 32 // cColorBits
	pfd[16] = 8 // cAlphaBits
	return pfd
}

func bindDC(hwnd uintptr) (uintptr, error) {
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return 0, fmt.Errorf("%w: GetDC", ErrUnavailable)
	}
	cur, _, _ := procGetPixelFormat.Call(hdc)
	if cur != 0 {
		return hdc, nil
	}
	pfd := pixelFormat()
	format, _, _ := procChoosePixelFormat.Call(hdc, uintptr(unsafe.Pointer(&pfd[0])))
	if format == 0 {
		procReleaseDC.Call(hwnd, hdc)
		runtime.KeepAlive(pfd)
		return 0, fmt.Errorf("%w: ChoosePixelFormat", ErrUnavailable)
	}
	ok, _, callErr := procSetPixelFormat.Call(hdc, format, uintptr(unsafe.Pointer(&pfd[0])))
	runtime.KeepAlive(pfd)
	if ok == 0 {
		procReleaseDC.Call(hwnd, hdc)
		return 0, fmt.Errorf("%w: SetPixelFormat %v", ErrUnavailable, callErr)
	}
	return hdc, nil
}

func releaseWindow(hwnd, hdc uintptr, own bool) {
	if hwnd != 0 && hdc != 0 {
		procReleaseDC.Call(hwnd, hdc)
	}
	if own && hwnd != 0 {
		procDestroyWindow.Call(hwnd)
	}
}

func hiddenWindow() (uintptr, error) {
	if err := ensureClass(); err != nil {
		return 0, err
	}
	title, err := syscall.UTF16PtrFromString("lewkit-opengl")
	if err != nil {
		return 0, err
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsPopup,
		0, 0, 1, 1,
		0, 0, inst, 0,
	)
	runtime.KeepAlive(title)
	if hwnd == 0 {
		return 0, fmt.Errorf("%w: CreateWindowExW %v", ErrUnavailable, callErr)
	}
	return hwnd, nil
}

func badProc(p uintptr) bool {
	return p == 0 || p == 1 || p == 2 || p == 3 || p == ^uintptr(0) || p == 0xFFFFFFFF
}

func lookupWGL(name string) uintptr {
	buf := native.CString(name)
	p, _, _ := procWglGetProcAddress.Call(uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(buf)
	if badProc(p) {
		return 0
	}
	return p
}

func createWGL(hwnd uintptr, own bool, major, minor int32) (*wglContext, error) {
	hdc, err := bindDC(hwnd)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*wglContext, error) {
		if hwnd != 0 && hdc != 0 {
			procReleaseDC.Call(hwnd, hdc)
		}
		return nil, err
	}
	legacy, _, _ := procWglCreateContext.Call(hdc)
	if legacy == 0 {
		return fail(fmt.Errorf("%w: wglCreateContext", ErrUnavailable))
	}
	if r, _, _ := procWglMakeCurrent.Call(hdc, legacy); r == 0 {
		procWglDeleteContext.Call(legacy)
		return fail(fmt.Errorf("%w: wglMakeCurrent", ErrUnavailable))
	}
	createAddr := lookupWGL("wglCreateContextAttribsARB")
	if createAddr == 0 {
		procWglMakeCurrent.Call(0, 0)
		procWglDeleteContext.Call(legacy)
		return fail(fmt.Errorf("%w: wglCreateContextAttribsARB", ErrUnavailable))
	}
	var create func(hdc, share, attribs uintptr) uintptr
	native.Register(&create, createAddr)
	attribs := []int32{wglContextMajor, major, wglContextMinor, minor, wglContextMask, wglCoreProfile, 0}
	ctx := create(hdc, 0, uintptr(unsafe.Pointer(&attribs[0])))
	runtime.KeepAlive(attribs)
	procWglMakeCurrent.Call(0, 0)
	procWglDeleteContext.Call(legacy)
	if ctx == 0 {
		return fail(fmt.Errorf("%w: core context %d.%d", ErrUnavailable, major, minor))
	}
	if r, _, _ := procWglMakeCurrent.Call(hdc, ctx); r == 0 {
		procWglDeleteContext.Call(ctx)
		return fail(fmt.Errorf("%w: core make current", ErrUnavailable))
	}
	procWglMakeCurrent.Call(0, 0)
	glLib, err := native.Open("opengl32.dll", 0)
	if err != nil {
		procWglDeleteContext.Call(ctx)
		return fail(err)
	}
	var getProc func(uintptr) uintptr
	if addr := wglProcAddress(); addr != 0 {
		native.Register(&getProc, addr)
	}
	return &wglContext{hwnd: hwnd, hdc: hdc, ctx: ctx, own: own, glLib: glLib, getProc: getProc}, nil
}

var (
	wglProcOnce sync.Once
	wglProcAddr uintptr
)

func wglProcAddress() uintptr {
	wglProcOnce.Do(func() {
		lib, err := native.Open("opengl32.dll", 0)
		if err != nil {
			return
		}
		wglProcAddr, _ = native.Symbol(lib, "wglGetProcAddress")
	})
	return wglProcAddr
}

func openWGL(hwnd uintptr, own bool, major, minor int32) (*wglContext, error) {
	ctx, err := createWGL(hwnd, own, major, minor)
	if err == nil {
		return ctx, nil
	}
	if major >= 4 {
		if own {
			procDestroyWindow.Call(hwnd)
		}
		return nil, err
	}
	ctx, err2 := createWGL(hwnd, own, 4, 3)
	if err2 != nil {
		if own {
			procDestroyWindow.Call(hwnd)
		}
		return nil, err
	}
	return ctx, nil
}

func (c *wglContext) Make() error {
	if c == nil || c.ctx == 0 || c.hdc == 0 {
		return ErrClosed
	}
	r, _, _ := procWglMakeCurrent.Call(c.hdc, c.ctx)
	if r == 0 {
		return fmt.Errorf("%w: wglMakeCurrent", ErrLost)
	}
	return nil
}

func (c *wglContext) Unmake() {
	if c == nil {
		return
	}
	procWglMakeCurrent.Call(0, 0)
}

func (c *wglContext) Swap() error {
	if c == nil || c.own || c.hdc == 0 {
		return nil
	}
	r, _, _ := procSwapBuffers.Call(c.hdc)
	if r == 0 {
		return fmt.Errorf("%w: SwapBuffers", ErrLost)
	}
	return nil
}

func (c *wglContext) Destroy() {
	if c == nil {
		return
	}
	c.Unmake()
	if c.ctx != 0 {
		procWglDeleteContext.Call(c.ctx)
		c.ctx = 0
	}
	releaseWindow(c.hwnd, c.hdc, c.own)
	c.hdc = 0
	if c.own {
		c.hwnd = 0
	}
}

func (c *wglContext) GLES() bool { return false }

func (c *wglContext) Proc(name string) uintptr {
	if c == nil {
		return 0
	}
	buf := native.CString(name)
	if c.getProc != nil {
		p := c.getProc(uintptr(unsafe.Pointer(&buf[0])))
		runtime.KeepAlive(buf)
		if !badProc(p) {
			return p
		}
	}
	runtime.KeepAlive(buf)
	if c.glLib != 0 {
		p, err := native.Symbol(c.glLib, name)
		if err == nil && p != 0 {
			return p
		}
	}
	return 0
}

func (c *wglContext) Retarget(native uintptr, _, _ int) error {
	if c == nil || c.own {
		return nil
	}
	if native == 0 {
		return ErrLost
	}
	if native == c.hwnd {
		return nil
	}
	hdc, err := bindDC(native)
	if err != nil {
		return err
	}
	if r, _, _ := procWglMakeCurrent.Call(hdc, c.ctx); r == 0 {
		procReleaseDC.Call(native, hdc)
		return fmt.Errorf("%w: retarget", ErrLost)
	}
	if c.hwnd != 0 && c.hdc != 0 {
		procReleaseDC.Call(c.hwnd, c.hdc)
	}
	c.hwnd = native
	c.hdc = hdc
	return nil
}

// OpenNative attaches a screen to an HWND. The window is not destroyed.
func OpenNative(kind int, a, _ uintptr, width, height int) (*Screen, error) {
	if kind != kindWin32 || a == 0 {
		return nil, ErrUnavailable
	}
	ctx, err := openWGL(a, false, 3, 3)
	if err != nil {
		return nil, err
	}
	return attach(ctx, width, height, false, a)
}

// OpenOffscreen draws into a framebuffer on a hidden window.
func OpenOffscreen(width, height int) (*Screen, error) {
	hwnd, err := hiddenWindow()
	if err != nil {
		return nil, err
	}
	ctx, err := openWGL(hwnd, true, 3, 3)
	if err != nil {
		return nil, err
	}
	return attach(ctx, width, height, true, 0)
}

// OpenDevice opens a GL 4.3 compute context. It does not show a window.
func OpenDevice() (*Device, error) {
	hwnd, err := hiddenWindow()
	if err != nil {
		return nil, err
	}
	ctx, err := createWGL(hwnd, true, 4, 3)
	if err != nil {
		procDestroyWindow.Call(hwnd)
		return nil, err
	}
	return openCompute(ctx)
}

var (
	presentProbe = native.Once(func() error {
		screen, err := OpenOffscreen(2, 2)
		if err != nil {
			return err
		}
		defer screen.Close()
		if err := screen.Draw(nil, nil, nil, 2, 2); err != nil {
			return err
		}
		pix, err := screen.Read()
		if err != nil {
			return err
		}
		if len(pix) < 4 || pix[0] != 0 || pix[1] != 0 || pix[2] != 0 || pix[3] != 255 {
			return ErrUnavailable
		}
		return nil
	})
	computeProbe = native.Once(func() error {
		device, err := OpenDevice()
		if err != nil {
			return err
		}
		defer device.Close()
		return probeCompute(device)
	})
)

// Available reports whether a present context can be created.
func Available() error { return presentProbe() }

// ComputeAvailable reports whether a compute context can dispatch.
func ComputeAvailable() error { return computeProbe() }
