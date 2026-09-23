//go:build darwin

package vulkan

import (
	"fmt"

	"github.com/ebitengine/purego/objc"
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
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	app.Send(objc.RegisterName("setActivationPolicy:"), 0)
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
	view.Send(objc.RegisterName("setWantsLayer:"), true)
	layer := objc.ID(objc.GetClass("CAMetalLayer")).Send(objc.RegisterName("alloc"))
	layer = layer.Send(objc.RegisterName("init"))
	if layer == 0 {
		wnd.Send(objc.RegisterName("close"))
		return nil, fmt.Errorf("%w: CAMetalLayer", ErrUnavailable)
	}
	layer.Send(objc.RegisterName("setDrawableSize:"), nsSize{Width: float64(width), Height: float64(height)})
	view.Send(objc.RegisterName("setLayer:"), layer)
	wnd.Send(objc.RegisterName("makeKeyAndOrderFront:"), objc.ID(0))
	app.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
	return &metalHost{wnd: wnd, layer: layer}, nil
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
