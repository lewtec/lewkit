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
	screen     *Screen
	wnd        objc.ID
	view       objc.ID
	layer      objc.ID
	lastW      int
	lastH      int
	closed     bool
	borrowed   bool
	createSurf func(inst uintptr, info *metalSurfaceInfo, alloc uintptr, surface *uint64) int32
}

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

func attachHost(s *Screen, kind int, a, b uintptr) (hostSurface, error) {
	if kind != 3 || a == 0 {
		return nil, ErrUnavailable
	}
	_ = b
	var host hostSurface
	var err error
	thread.Do(func() {
		host, err = attachMetalView(s, objc.ID(a))
	})
	return host, err
}

func attachMetalView(s *Screen, view objc.ID) (hostSurface, error) {
	if _, err := nsApp(); err != nil {
		return nil, err
	}
	layer := objc.ID(objc.GetClass("CAMetalLayer")).Send(objc.RegisterName("alloc"))
	layer = layer.Send(objc.RegisterName("init"))
	if layer == 0 {
		return nil, fmt.Errorf("%w: CAMetalLayer", ErrUnavailable)
	}
	rect := metalBounds(view)
	w := rect.Size.Width
	h := rect.Size.Height
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	layer.Send(objc.RegisterName("setContentsScale:"), 1.0)
	layer.Send(objc.RegisterName("setDrawableSize:"), nsSize{Width: w, Height: h})
	layer.Send(objc.RegisterName("setDisplaySyncEnabled:"), true)
	view.Send(objc.RegisterName("setLayer:"), layer)
	view.Send(objc.RegisterName("setWantsLayer:"), true)
	host := &metalHost{screen: s, view: view, layer: layer, lastW: int(w + 0.5), lastH: int(h + 0.5), borrowed: true}
	trackMetal(host)
	return host, nil
}

func openHost(screen *Screen, width, height int, title string) (hostSurface, error) {
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
		host, err = openMetalWindow(screen, width, height, title)
	})
	return host, err
}

func openMetalWindow(screen *Screen, width, height int, title string) (hostSurface, error) {
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
	layer.Send(objc.RegisterName("setDisplaySyncEnabled:"), true)
	view.Send(objc.RegisterName("setLayer:"), layer)
	view.Send(objc.RegisterName("setWantsLayer:"), true)
	wnd.Send(objc.RegisterName("center"))
	wnd.Send(objc.RegisterName("orderFrontRegardless"))
	wnd.Send(objc.RegisterName("makeKeyAndOrderFront:"), objc.ID(0))
	wnd.Send(objc.RegisterName("display"))
	wnd.Send(objc.RegisterName("setAcceptsMouseMovedEvents:"), true)
	app.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
	host := &metalHost{screen: screen, wnd: wnd, layer: layer, lastW: width, lastH: height}
	trackMetal(host)
	pollDarwin()
	return host, nil
}

func (h *metalHost) poll() {}

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
		thread.OnIdle(pollDarwin)
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
	if h == nil {
		return
	}
	untrackMetal(h)
	if h.borrowed || h.wnd == 0 {
		h.wnd = 0
		h.layer = 0
		return
	}
	wnd := h.wnd
	h.wnd = 0
	h.layer = 0
	thread.Do(func() {
		wnd.Send(objc.RegisterName("close"))
	})
}

var (
	metalMu sync.Mutex
	metals  []*metalHost
)

func trackMetal(h *metalHost) {
	metalMu.Lock()
	metals = append(metals, h)
	metalMu.Unlock()
}

func untrackMetal(h *metalHost) {
	metalMu.Lock()
	for i, item := range metals {
		if item == h {
			metals = append(metals[:i], metals[i+1:]...)
			break
		}
	}
	metalMu.Unlock()
}

func pollDarwin() {
	if !thread.ProcessMain() {
		return
	}
	pumpApp()
	dispatchMetalEvents()
	metalMu.Lock()
	hosts := append([]*metalHost(nil), metals...)
	metalMu.Unlock()
	for _, h := range hosts {
		h.note()
	}
}

func dispatchMetalEvents() {
	date := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("distantPast"))
	if date == 0 {
		return
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	mode := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), "NSDefaultRunLoopMode")
	if app == 0 || mode == 0 {
		return
	}
	ev := app.Send(objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:"), ^uintptr(0), date, mode, true)
	if ev == 0 {
		return
	}
	deliverMetal(ev)
	app.Send(objc.RegisterName("sendEvent:"), ev)
}

func deliverMetal(ev objc.ID) {
	nsw := ev.Send(objc.RegisterName("window"))
	metalMu.Lock()
	var host *metalHost
	for _, h := range metals {
		if h.wnd == nsw {
			host = h
			break
		}
	}
	metalMu.Unlock()
	if host == nil || host.screen == nil {
		return
	}
	typ := int(ev.Send(objc.RegisterName("type")))
	pos := metalPos(host, ev)
	switch typ {
	case 1, 3, 25:
		host.screen.emit(Input{Kind: InputPointer, X: pos.X, Y: pos.Y, Button: metalButton(int(ev.Send(objc.RegisterName("buttonNumber")))), Pressed: true})
	case 2, 4, 26:
		host.screen.emit(Input{Kind: InputPointer, X: pos.X, Y: pos.Y, Button: metalButton(int(ev.Send(objc.RegisterName("buttonNumber"))))})
	case 5, 6, 7, 27:
		host.screen.emit(Input{Kind: InputPointer, X: pos.X, Y: pos.Y})
	case 22:
		dx, dy := metalDelta(ev, "deltaX"), metalDelta(ev, "deltaY")
		host.screen.emit(Input{Kind: InputScroll, X: pos.X, Y: pos.Y, DX: int(dx), DY: int(-dy)})
	case 10, 11:
		host.screen.emit(Input{Kind: InputKey, X: pos.X, Y: pos.Y, Code: uint32(ev.Send(objc.RegisterName("keyCode"))), Pressed: typ == 10, Repeat: ev.Send(objc.RegisterName("isARepeat")) != 0, Rune: metalRune(ev.Send(objc.RegisterName("characters"))), Mod: metalMod(uintptr(ev.Send(objc.RegisterName("modifierFlags"))))})
	}
}

func (h *metalHost) fitLayer() {
	if h == nil || h.layer == 0 || h.view == 0 {
		return
	}
	rect := metalBounds(h.view)
	w := int(rect.Size.Width + 0.5)
	hgt := int(rect.Size.Height + 0.5)
	if w < 1 || hgt < 1 || (w == h.lastW && hgt == h.lastH) {
		return
	}
	h.lastW, h.lastH = w, hgt
	h.layer.Send(objc.RegisterName("setDrawableSize:"), nsSize{Width: float64(w), Height: float64(hgt)})
}

func (h *metalHost) note() {
	if h == nil || h.screen == nil || h.closed {
		return
	}
	if h.borrowed {
		h.fitLayer()
		return
	}
	if h.wnd == 0 {
		return
	}
	if h.wnd.Send(objc.RegisterName("isVisible")) == 0 {
		h.closed = true
		h.screen.emit(Input{Kind: InputClose})
		return
	}
	view := h.wnd.Send(objc.RegisterName("contentView"))
	if view == 0 {
		return
	}
	rect := metalBounds(view)
	w := int(rect.Size.Width + 0.5)
	hgt := int(rect.Size.Height + 0.5)
	if w < 1 || hgt < 1 || (w == h.lastW && hgt == h.lastH) {
		return
	}
	h.lastW, h.lastH = w, hgt
	h.layer.Send(objc.RegisterName("setDrawableSize:"), nsSize{Width: float64(w), Height: float64(hgt)})
	h.screen.emit(Input{Kind: InputResize, X: w, Y: hgt})
}

func metalButton(n int) int {
	switch n {
	case 1:
		return 2
	case 2:
		return 3
	default:
		return 1
	}
}

func metalMod(flags uintptr) uint32 {
	var m uint32
	if flags&(1<<17) != 0 {
		m |= 1
	}
	if flags&(1<<18) != 0 {
		m |= 2
	}
	if flags&(1<<19) != 0 {
		m |= 4
	}
	if flags&(1<<20) != 0 {
		m |= 8
	}
	return m
}

var (
	metalLoc     func(objc.ID, objc.SEL) nsPoint
	metalDeltaF  func(objc.ID, objc.SEL) float64
	metalBoundsF func(objc.ID, objc.SEL) nsRect
	objcSend     uintptr
)

func metalMsgSend() {
	if objcSend != 0 {
		return
	}
	addr, err := native.Symbol(mustObjc(), "objc_msgSend")
	if err != nil {
		return
	}
	objcSend = addr
	native.Register(&metalLoc, objcSend)
	native.Register(&metalDeltaF, objcSend)
	native.Register(&metalBoundsF, objcSend)
}

func mustObjc() uintptr {
	lib, err := native.Open("/usr/lib/libobjc.A.dylib", native.Global|native.Lazy)
	if err != nil {
		return 0
	}
	return lib
}

func metalPos(h *metalHost, ev objc.ID) imagePoint {
	metalMsgSend()
	if metalLoc == nil || h.wnd == 0 {
		return imagePoint{}
	}
	loc := metalLoc(ev, objc.RegisterName("locationInWindow"))
	view := h.wnd.Send(objc.RegisterName("contentView"))
	rect := metalBounds(view)
	return imagePoint{X: int(loc.X + 0.5), Y: int(rect.Size.Height - loc.Y + 0.5)}
}

func metalBounds(view objc.ID) nsRect {
	metalMsgSend()
	if metalBoundsF == nil || view == 0 {
		return nsRect{}
	}
	return metalBoundsF(view, objc.RegisterName("bounds"))
}

func metalDelta(ev objc.ID, name string) float64 {
	metalMsgSend()
	if metalDeltaF == nil {
		return 0
	}
	return metalDeltaF(ev, objc.RegisterName(name))
}

func metalRune(ns objc.ID) rune {
	if ns == 0 {
		return 0
	}
	p := ns.Send(objc.RegisterName("UTF8String"))
	if p == 0 {
		return 0
	}
	n := 0
	for *(*byte)(unsafe.Pointer(uintptr(p) + uintptr(n))) != 0 && n < 16 {
		n++
	}
	s := string(unsafe.Slice((*byte)(unsafe.Pointer(uintptr(p))), n))
	for _, r := range s {
		return r
	}
	return 0
}

type imagePoint struct{ X, Y int }
