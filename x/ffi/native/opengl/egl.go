//go:build linux

package opengl

import (
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	eglPlatformX11         = 0x31D5
	eglPlatformSurfaceless = 0x31DD
	eglOpenglAPI           = 0x30A2
	eglOpenglESAPI         = 0x30A0
	eglOpenglBit           = 0x0008
	eglOpenglES3Bit        = 0x00000040
	eglSurfaceType         = 0x3033
	eglPbufferBit          = 0x0001
	eglWindowBit           = 0x0004
	eglRenderableType      = 0x3040
	eglRedSize             = 0x3024
	eglGreenSize           = 0x3023
	eglBlueSize            = 0x3022
	eglAlphaSize           = 0x3021
	eglNone                = 0x3038
	eglWidth               = 0x3057
	eglHeight              = 0x3056
	eglContextMajor        = 0x3098
	eglContextMinor        = 0x30FB
	eglContextProfile      = 0x30FD
	eglCoreProfile         = 0x00000001
)

type eglLib struct {
	getDisplay     func(uintptr) uintptr
	getPlatform    func(uint32, uintptr, uintptr) uintptr
	initialize     func(uintptr, *int32, *int32) uint32
	bindAPI        func(uint32) uint32
	chooseConfig   func(uintptr, *int32, *uintptr, int32, *int32) uint32
	createContext  func(uintptr, uintptr, uintptr, *int32) uintptr
	createWindow   func(uintptr, uintptr, uintptr, *int32) uintptr
	createPbuffer  func(uintptr, uintptr, *int32) uintptr
	makeCurrent    func(uintptr, uintptr, uintptr, uintptr) uint32
	swap           func(uintptr, uintptr) uint32
	destroyContext func(uintptr, uintptr) uint32
	destroySurface func(uintptr, uintptr) uint32
	getProc        func(*byte) uintptr
	getError       func() int32
}

type eglContext struct {
	lib     *eglLib
	dpy     uintptr
	ctx     uintptr
	surf    uintptr
	cfg     uintptr
	glLib   uintptr
	win     uintptr
	gles    bool
	replace bool
}

var loadEGL = native.Singleton(func() (*eglLib, error) {
	lib, err := native.OpenChain(native.Now|native.Global, "libEGL.so.1", "libEGL.so")
	if err != nil {
		return nil, err
	}
	e := &eglLib{}
	binds := []struct {
		name string
		fn   any
	}{
		{"eglGetDisplay", &e.getDisplay},
		{"eglInitialize", &e.initialize},
		{"eglBindAPI", &e.bindAPI},
		{"eglChooseConfig", &e.chooseConfig},
		{"eglCreateContext", &e.createContext},
		{"eglCreateWindowSurface", &e.createWindow},
		{"eglCreatePbufferSurface", &e.createPbuffer},
		{"eglMakeCurrent", &e.makeCurrent},
		{"eglSwapBuffers", &e.swap},
		{"eglDestroyContext", &e.destroyContext},
		{"eglDestroySurface", &e.destroySurface},
		{"eglGetProcAddress", &e.getProc},
		{"eglGetError", &e.getError},
	}
	for _, item := range binds {
		if err := native.Bind(lib, item.name, item.fn); err != nil {
			return nil, err
		}
	}
	_ = native.Bind(lib, "eglGetPlatformDisplay", &e.getPlatform)
	return e, nil
})

func preloadGL(gles bool) (uintptr, error) {
	if gles {
		return native.OpenChain(native.Now|native.Global, "libGLESv2.so.2", "libGLESv2.so")
	}
	return native.OpenChain(native.Now|native.Global, "libOpenGL.so.0", "libGL.so.1")
}

// openEGL opens a desktop or ES context. nativeDisplay and window are zero
// for an offscreen context. replace is set when Adopt may swap the window.
func openEGL(nativeDisplay, window uintptr, gles, compute, replace bool) (*eglContext, error) {
	lib, err := loadEGL()
	if err != nil {
		return nil, err
	}
	glLib, err := preloadGL(gles)
	if err != nil {
		return nil, err
	}
	api := uint32(eglOpenglAPI)
	major, minor := int32(3), int32(3)
	if compute && !gles {
		major, minor = 4, 3
	}
	if gles {
		api = eglOpenglESAPI
		major = 3
		minor = 0
		if compute {
			minor = 1
		}
	}
	var last error
	for _, dpy := range eglDisplays(lib, nativeDisplay) {
		var vmaj, vmin int32
		if lib.initialize(dpy, &vmaj, &vmin) == 0 {
			last = fmt.Errorf("%w: eglInitialize 0x%x", ErrUnavailable, lib.getError())
			continue
		}
		if lib.bindAPI(api) == 0 {
			last = fmt.Errorf("%w: eglBindAPI 0x%x", ErrUnavailable, lib.getError())
			continue
		}
		cfg, err := eglConfig(lib, dpy, gles, window != 0)
		if err != nil {
			last = err
			continue
		}
		ctx, err := eglCreate(lib, dpy, cfg, gles, major, minor)
		if err != nil {
			last = err
			continue
		}
		out := &eglContext{lib: lib, dpy: dpy, ctx: ctx, cfg: cfg, glLib: glLib, gles: gles, win: window, replace: replace}
		if window != 0 {
			surf, err := eglWindow(lib, dpy, cfg, window)
			if err != nil {
				lib.destroyContext(dpy, ctx)
				last = err
				continue
			}
			if lib.makeCurrent(dpy, surf, surf, ctx) == 0 {
				lib.destroySurface(dpy, surf)
				lib.destroyContext(dpy, ctx)
				last = fmt.Errorf("%w: eglMakeCurrent 0x%x", ErrUnavailable, lib.getError())
				continue
			}
			lib.makeCurrent(dpy, 0, 0, 0)
			out.surf = surf
			return out, nil
		}
		if lib.makeCurrent(dpy, 0, 0, ctx) != 0 {
			lib.makeCurrent(dpy, 0, 0, 0)
			return out, nil
		}
		surf, err := eglPbuffer(lib, dpy, cfg)
		if err != nil {
			lib.destroyContext(dpy, ctx)
			last = err
			continue
		}
		out.surf = surf
		if lib.makeCurrent(dpy, surf, surf, ctx) == 0 {
			lib.destroySurface(dpy, surf)
			lib.destroyContext(dpy, ctx)
			last = fmt.Errorf("%w: eglMakeCurrent 0x%x", ErrUnavailable, lib.getError())
			continue
		}
		lib.makeCurrent(dpy, 0, 0, 0)
		return out, nil
	}
	if last == nil {
		last = ErrUnavailable
	}
	return nil, last
}

func eglDisplays(lib *eglLib, native uintptr) []uintptr {
	var out []uintptr
	add := func(d uintptr) {
		if d == 0 {
			return
		}
		for _, have := range out {
			if have == d {
				return
			}
		}
		out = append(out, d)
	}
	if native != 0 {
		if lib.getPlatform != nil {
			add(lib.getPlatform(eglPlatformX11, native, 0))
		}
		add(lib.getDisplay(native))
		return out
	}
	add(lib.getDisplay(0))
	if lib.getPlatform != nil {
		add(lib.getPlatform(eglPlatformSurfaceless, 0, 0))
	}
	return out
}

func eglConfig(lib *eglLib, dpy uintptr, gles, window bool) (uintptr, error) {
	render := int32(eglOpenglBit)
	if gles {
		render = eglOpenglES3Bit
	}
	surface := int32(eglPbufferBit)
	if window {
		surface = eglWindowBit
	}
	attribs := []int32{
		eglSurfaceType, surface,
		eglRenderableType, render,
		eglRedSize, 8,
		eglGreenSize, 8,
		eglBlueSize, 8,
		eglAlphaSize, 8,
		eglNone,
	}
	var cfg uintptr
	var n int32
	if lib.chooseConfig(dpy, &attribs[0], &cfg, 1, &n) == 0 || n < 1 || cfg == 0 {
		return 0, fmt.Errorf("%w: eglChooseConfig 0x%x", ErrUnavailable, lib.getError())
	}
	return cfg, nil
}

func eglCreate(lib *eglLib, dpy, cfg uintptr, gles bool, major, minor int32) (uintptr, error) {
	attribs := []int32{eglContextMajor, major, eglContextMinor, minor, eglNone}
	if !gles {
		attribs = []int32{
			eglContextMajor, major,
			eglContextMinor, minor,
			eglContextProfile, eglCoreProfile,
			eglNone,
		}
	}
	ctx := lib.createContext(dpy, cfg, 0, &attribs[0])
	if ctx == 0 {
		return 0, fmt.Errorf("%w: eglCreateContext 0x%x", ErrUnavailable, lib.getError())
	}
	return ctx, nil
}

func eglWindow(lib *eglLib, dpy, cfg, win uintptr) (uintptr, error) {
	surf := lib.createWindow(dpy, cfg, win, nil)
	if surf == 0 {
		return 0, fmt.Errorf("%w: eglCreateWindowSurface 0x%x", ErrUnavailable, lib.getError())
	}
	return surf, nil
}

func eglPbuffer(lib *eglLib, dpy, cfg uintptr) (uintptr, error) {
	attribs := []int32{eglWidth, 1, eglHeight, 1, eglNone}
	surf := lib.createPbuffer(dpy, cfg, &attribs[0])
	if surf == 0 {
		return 0, fmt.Errorf("%w: eglCreatePbufferSurface 0x%x", ErrUnavailable, lib.getError())
	}
	return surf, nil
}

func (c *eglContext) Make() error {
	if c == nil || c.lib == nil || c.ctx == 0 {
		return ErrClosed
	}
	draw, read := c.surf, c.surf
	if c.lib.makeCurrent(c.dpy, draw, read, c.ctx) == 0 {
		return fmt.Errorf("%w: eglMakeCurrent 0x%x", ErrLost, c.lib.getError())
	}
	return nil
}

func (c *eglContext) Unmake() {
	if c == nil || c.lib == nil {
		return
	}
	c.lib.makeCurrent(c.dpy, 0, 0, 0)
}

func (c *eglContext) Swap() error {
	if c == nil || c.lib == nil || c.surf == 0 || c.win == 0 {
		return nil
	}
	if c.lib.swap(c.dpy, c.surf) == 0 {
		return fmt.Errorf("%w: eglSwapBuffers 0x%x", ErrLost, c.lib.getError())
	}
	return nil
}

func (c *eglContext) Destroy() {
	if c == nil || c.lib == nil {
		return
	}
	c.Unmake()
	if c.surf != 0 {
		c.lib.destroySurface(c.dpy, c.surf)
		c.surf = 0
	}
	if c.ctx != 0 {
		c.lib.destroyContext(c.dpy, c.ctx)
		c.ctx = 0
	}
}

func (c *eglContext) GLES() bool { return c != nil && c.gles }

func (c *eglContext) Proc(name string) uintptr {
	if c == nil || c.lib == nil {
		return 0
	}
	buf := native.CString(name)
	if c.lib.getProc != nil {
		if p := c.lib.getProc(&buf[0]); p != 0 {
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

func (c *eglContext) Retarget(native uintptr, _, _ int) error {
	if c == nil || !c.replace {
		return nil
	}
	if native == 0 {
		return ErrLost
	}
	c.Unmake()
	if c.surf != 0 {
		c.lib.destroySurface(c.dpy, c.surf)
		c.surf = 0
	}
	surf, err := eglWindow(c.lib, c.dpy, c.cfg, native)
	if err != nil {
		return err
	}
	c.surf = surf
	c.win = native
	return c.Make()
}

func openEGLOffscreen(compute bool) (glContext, error) {
	if !compute {
		if ctx, err := openEGL(0, 0, false, false, false); err == nil {
			return ctx, nil
		}
		return openEGL(0, 0, true, false, false)
	}
	if ctx, err := openEGL(0, 0, false, true, false); err == nil {
		return ctx, nil
	}
	return openEGL(0, 0, true, true, false)
}
