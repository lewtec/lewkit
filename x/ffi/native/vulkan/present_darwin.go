//go:build darwin

package vulkan

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/thread"
)

const extHostSurface = "VK_EXT_metal_surface"

func surfaceExtensions() []string { return []string{extHostSurface} }

func loadHost(*wsi, *Device) error { return nil }

type metalSurfaceInfo struct {
	sType int32
	pNext uintptr
	flags uint32
	layer uintptr
}

type metalHost struct {
	wnd        objc.ID
	layer      objc.ID
	createSurf func(inst uintptr, info *metalSurfaceInfo, alloc uintptr, surface *uint64) int32
}

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

func openHost(width, height int, title string) (hostSurface, error) {
	if !thread.Bound() {
		return nil, fmt.Errorf("%w: main thread", ErrUnavailable)
	}
	var host hostSurface
	var err error
	thread.Do(func() {
		if !thread.ProcessMain() {
			err = fmt.Errorf("%w: NSWindow requires the main thread", ErrUnavailable)
			return
		}
		host, err = openMetalWindow(width, height, title)
	})
	return host, err
}

func openMetalWindow(width, height int, title string) (hostSurface, error) {
	app, err := nsApp()
	if err != nil {
		return nil, err
	}
	rect := nsRect{Size: nsSize{Width: float64(width), Height: float64(height)}}
	wnd := objc.ID(objc.GetClass("NSWindow")).Send(objc.RegisterName("alloc"))
	wnd = wnd.Send(objc.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect, uintptr(1|2|4|8), uintptr(2), false)
	if wnd == 0 {
		return nil, fmt.Errorf("%w: ns window", ErrUnavailable)
	}
	if title != "" {
		ns := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), title)
		wnd.Send(objc.RegisterName("setTitle:"), ns)
	}
	view := wnd.Send(objc.RegisterName("contentView"))
	layer := objc.ID(objc.GetClass("CAMetalLayer")).Send(objc.RegisterName("alloc"))
	layer = layer.Send(objc.RegisterName("init"))
	if layer == 0 {
		wnd.Send(objc.RegisterName("close"))
		return nil, fmt.Errorf("%w: CAMetalLayer", ErrUnavailable)
	}
	layer.Send(objc.RegisterName("setContentsScale:"), 1.0)
	layer.Send(objc.RegisterName("setDrawableSize:"), nsSize{Width: float64(width), Height: float64(height)})
	view.Send(objc.RegisterName("setLayer:"), layer)
	view.Send(objc.RegisterName("setWantsLayer:"), true)
	wnd.Send(objc.RegisterName("center"))
	wnd.Send(objc.RegisterName("orderFrontRegardless"))
	wnd.Send(objc.RegisterName("makeKeyAndOrderFront:"), objc.ID(0))
	wnd.Send(objc.RegisterName("display"))
	app.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
	pumpApp()
	return &metalHost{wnd: wnd, layer: layer}, nil
}

var appOnce sync.Once

func nsApp() (objc.ID, error) {
	var app objc.ID
	var err error
	appOnce.Do(func() {
		if _, openErr := native.Open("/System/Library/Frameworks/Cocoa.framework/Cocoa", native.Global|native.Lazy); openErr != nil {
			err = fmt.Errorf("%w: cocoa: %w", ErrUnavailable, openErr)
			return
		}
		app = objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
		app.Send(objc.RegisterName("setActivationPolicy:"), 0)
		app.Send(objc.RegisterName("finishLaunching"))
		thread.OnIdle(pumpApp)
	})
	if err != nil {
		return 0, err
	}
	if app == 0 {
		app = objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	}
	return app, nil
}

var (
	cfOnce             sync.Once
	cfRunLoopRunInMode func(mode uintptr, seconds float64, returnAfter bool) int32
	cfDefaultMode      uintptr
)

func pumpApp() {
	if !thread.ProcessMain() {
		return
	}
	cfOnce.Do(func() {
		lib, err := native.Open("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", native.Global|native.Lazy)
		if err != nil {
			return
		}
		native.Func(lib, "CFRunLoopRunInMode", &cfRunLoopRunInMode)
		addr, err := native.Symbol(lib, "kCFRunLoopDefaultMode")
		if err != nil || addr == 0 {
			return
		}
		cfDefaultMode = *(*uintptr)(unsafe.Pointer(addr))
	})
	if cfRunLoopRunInMode == nil || cfDefaultMode == 0 {
		return
	}
	// Zero seconds: drain what AppKit already queued and return. A nil
	// NSDate in nextEventMatchingMask blocks the main thread forever.
	cfRunLoopRunInMode(cfDefaultMode, 0, false)
}

func (h *metalHost) create(d *Device, w *wsi) (uint64, error) {
	_ = w
	if err := d.api.bind(d.api.getInstanceProcAddr, d.inst, "vkCreateMetalSurfaceEXT", &h.createSurf); err != nil {
		return 0, err
	}
	info := metalSurfaceInfo{sType: 1000217000, layer: uintptr(h.layer)}
	var surface uint64
	if err := check(h.createSurf(d.inst, &info, 0, &surface)); err != nil {
		return 0, fmt.Errorf("metal surface: %w", err)
	}
	return surface, nil
}

func (h *metalHost) destroy() {
	if h == nil || h.wnd == 0 {
		return
	}
	wnd := h.wnd
	h.wnd = 0
	h.layer = 0
	thread.Do(func() {
		wnd.Send(objc.RegisterName("close"))
	})
}
