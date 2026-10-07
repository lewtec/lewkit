//go:build linux && !android

package opengl

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	glxDrawableType = 0x8010
	glxRenderType   = 0x8011
	glxXRenderable  = 0x8012
	glxWindowBit    = 0x00000001
	glxRGBABit      = 0x00000001
	glxVisualID     = 0x800B
	glxDoubleBuffer = 5
	glxRedSize      = 8
	glxGreenSize    = 9
	glxBlueSize     = 10
	glxAlphaSize    = 11
	glxContextMajor = 0x2091
	glxContextMinor = 0x2092
	glxContextMask  = 0x9126
	glxCoreProfile  = 0x00000001
)

type glxContext struct {
	dpy     uintptr
	ctx     uintptr
	win     uintptr
	glLib   uintptr
	getProc func(*byte) uintptr
	make    func(uintptr, uintptr, uintptr) int32
	swapBuf func(uintptr, uintptr)
	destroy func(uintptr, uintptr)
}

func openGLX(dpy, win uintptr) (glContext, error) {
	if dpy == 0 || win == 0 {
		return nil, ErrUnavailable
	}
	glLib, err := native.OpenChain(native.Now|native.Global, "libGL.so.1")
	if err != nil {
		return nil, err
	}
	xLib, err := native.OpenChain(native.Now|native.Global, "libX11.so.6")
	if err != nil {
		return nil, err
	}
	var (
		defaultScreen func(uintptr) int32
		windowAttr    func(uintptr, uintptr, unsafe.Pointer) int32
		visualID      func(uintptr) uintptr
		xFree         func(uintptr) int32
		choose        func(uintptr, int32, *int32, *int32) uintptr
		getAttrib     func(uintptr, uintptr, int32, *int32) int32
		getProc       func(*byte) uintptr
		makeCurrent   func(uintptr, uintptr, uintptr) int32
		swap          func(uintptr, uintptr)
		destroy       func(uintptr, uintptr)
	)
	binds := []struct {
		lib  uintptr
		name string
		fn   any
	}{
		{xLib, "XDefaultScreen", &defaultScreen},
		{xLib, "XGetWindowAttributes", &windowAttr},
		{xLib, "XVisualIDFromVisual", &visualID},
		{xLib, "XFree", &xFree},
		{glLib, "glXChooseFBConfig", &choose},
		{glLib, "glXGetFBConfigAttrib", &getAttrib},
		{glLib, "glXGetProcAddress", &getProc},
		{glLib, "glXMakeCurrent", &makeCurrent},
		{glLib, "glXSwapBuffers", &swap},
		{glLib, "glXDestroyContext", &destroy},
	}
	for _, item := range binds {
		if err := native.Bind(item.lib, item.name, item.fn); err != nil {
			return nil, err
		}
	}
	var raw [256]byte
	if windowAttr(dpy, win, unsafe.Pointer(&raw[0])) == 0 {
		return nil, fmt.Errorf("%w: XGetWindowAttributes", ErrUnavailable)
	}
	visual := *(*uintptr)(unsafe.Pointer(&raw[24]))
	want := visualID(visual)
	screen := defaultScreen(dpy)
	attribs := []int32{
		glxXRenderable, 1,
		glxDrawableType, glxWindowBit,
		glxRenderType, glxRGBABit,
		glxRedSize, 8,
		glxGreenSize, 8,
		glxBlueSize, 8,
		glxAlphaSize, 8,
		glxDoubleBuffer, 1,
		0,
	}
	var n int32
	configs := choose(dpy, screen, &attribs[0], &n)
	if configs == 0 || n < 1 {
		return nil, fmt.Errorf("%w: glXChooseFBConfig", ErrUnavailable)
	}
	defer xFree(configs)
	var match uintptr
	for i := int32(0); i < n; i++ {
		cfg := *(*uintptr)(unsafe.Pointer(configs + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		var id int32
		if getAttrib(dpy, cfg, glxVisualID, &id) != 0 {
			continue
		}
		if uintptr(id) == want {
			match = cfg
			break
		}
	}
	if match == 0 {
		return nil, fmt.Errorf("%w: window visual", ErrUnavailable)
	}
	name := native.CString("glXCreateContextAttribsARB")
	createAddr := getProc(&name[0])
	runtime.KeepAlive(name)
	if createAddr == 0 {
		return nil, fmt.Errorf("%w: glXCreateContextAttribsARB", ErrUnavailable)
	}
	var create func(uintptr, uintptr, uintptr, int32, *int32) uintptr
	native.Register(&create, createAddr)
	profile := []int32{glxContextMajor, 3, glxContextMinor, 3, glxContextMask, glxCoreProfile, 0}
	ctx := create(dpy, match, 0, 1, &profile[0])
	if ctx == 0 {
		return nil, fmt.Errorf("%w: glXCreateContext", ErrUnavailable)
	}
	if makeCurrent(dpy, win, ctx) == 0 {
		destroy(dpy, ctx)
		return nil, fmt.Errorf("%w: glXMakeCurrent", ErrUnavailable)
	}
	makeCurrent(dpy, 0, 0)
	return &glxContext{
		dpy: dpy, ctx: ctx, win: win, glLib: glLib, getProc: getProc,
		make: makeCurrent, swapBuf: swap, destroy: destroy,
	}, nil
}

func (c *glxContext) Make() error {
	if c == nil || c.ctx == 0 || c.make == nil {
		return ErrClosed
	}
	if c.make(c.dpy, c.win, c.ctx) == 0 {
		return ErrLost
	}
	return nil
}

func (c *glxContext) Unmake() {
	if c == nil || c.make == nil {
		return
	}
	c.make(c.dpy, 0, 0)
}

func (c *glxContext) Swap() error {
	if c == nil || c.swapBuf == nil || c.win == 0 {
		return nil
	}
	c.swapBuf(c.dpy, c.win)
	return nil
}

func (c *glxContext) Destroy() {
	if c == nil || c.ctx == 0 {
		return
	}
	c.Unmake()
	if c.destroy != nil {
		c.destroy(c.dpy, c.ctx)
	}
	c.ctx = 0
}

func (c *glxContext) GLES() bool { return false }

func (c *glxContext) Proc(name string) uintptr {
	if c == nil {
		return 0
	}
	buf := native.CString(name)
	if c.getProc != nil {
		if p := c.getProc(&buf[0]); p != 0 {
			runtime.KeepAlive(buf)
			return p
		}
	}
	runtime.KeepAlive(buf)
	if c.glLib != 0 {
		p, err := native.Symbol(c.glLib, name)
		if err == nil {
			return p
		}
	}
	return 0
}
