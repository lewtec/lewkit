//go:build darwin

package cocoa

import (
	"context"
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/thread"
)

const (
	nsWindowStyleTitled          = 1 << 0
	nsWindowStyleClosable        = 1 << 1
	nsWindowStyleMiniaturizable  = 1 << 2
	nsWindowStyleResizable       = 1 << 3
	nsBackingBuffered            = 2
	nsApplicationActivateRegular = 0
)

type nsPoint struct {
	X, Y float64
}

type nsSize struct {
	Width, Height float64
}

type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

var (
	appOnce sync.Once
	appErr  error

	selContentView               = objc.RegisterName("contentView")
	selSetWantsLayer             = objc.RegisterName("setWantsLayer:")
	selLayer                     = objc.RegisterName("layer")
	selSetContents               = objc.RegisterName("setContents:")
	selSetContentsScale          = objc.RegisterName("setContentsScale:")
	selSetMagFilter              = objc.RegisterName("setMagnificationFilter:")
	selSetMinFilter              = objc.RegisterName("setMinificationFilter:")
	selSetContentsGravity        = objc.RegisterName("setContentsGravity:")
	selSetAllowsEdgeAntialiasing = objc.RegisterName("setAllowsEdgeAntialiasing:")
	selSetEdgeAntialiasingMask   = objc.RegisterName("setEdgeAntialiasingMask:")
	selSetTitle                  = objc.RegisterName("setTitle:")
	selMakeKeyAndOrderFront      = objc.RegisterName("makeKeyAndOrderFront:")
	selSetContentSize            = objc.RegisterName("setContentSize:")
	selClose                     = objc.RegisterName("close")
	selIsVisible                 = objc.RegisterName("isVisible")
	selBounds                    = objc.RegisterName("bounds")
	selBackingScaleFactor        = objc.RegisterName("backingScaleFactor")

	live             sync.Map // *win → struct{}
	selNextEvent     = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent     = objc.RegisterName("sendEvent:")
	selUpdateWindows = objc.RegisterName("updateWindows")
	selDistantPast   = objc.RegisterName("distantPast")
)

func startApp() error {
	if !thread.Bound() {
		return fmt.Errorf("cocoa: thread.Bind was not called from main")
	}
	appOnce.Do(func() {
		thread.Do(func() {
			if !thread.ProcessMain() {
				appErr = fmt.Errorf("cocoa: NSApplication is not on the process main thread")
				return
			}
			if _, err := purego.Dlopen("/System/Library/Frameworks/Cocoa.framework/Cocoa", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
				appErr = err
				return
			}
			if _, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
				appErr = err
				return
			}
			loadCG()
			app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
			app.Send(objc.RegisterName("setActivationPolicy:"), nsApplicationActivateRegular)
			thread.OnIdle(func() {
				pump(app)
				live.Range(func(k, _ any) bool {
					k.(*win).note()
					return true
				})
			})
		})
	})
	return appErr
}

func onApp(fn func()) {
	thread.Do(fn)
}

func pump(app objc.ID) {
	if !thread.ProcessMain() {
		return
	}
	date := objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
	mode := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), "kCFRunLoopDefaultMode")
	for {
		ev := app.Send(selNextEvent, ^uintptr(0), date, mode, true)
		if ev == 0 {
			break
		}
		app.Send(selSendEvent, ev)
	}
	app.Send(selUpdateWindows)
}

func (cdriver) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	if err := startApp(); err != nil {
		return nil, err
	}
	out := &win{Buffer: window.NewBuffer(w, h)}
	var openErr error
	onApp(func() {
		openErr = out.create(cfg.Title, w, h)
		if openErr == nil {
			if ww, hh := out.clientSize(); ww > 0 && hh > 0 {
				_ = out.Buffer.Resize(image.Pt(ww, hh))
			}
		}
	})
	if openErr != nil {
		return nil, openErr
	}
	live.Store(out, struct{}{})
	if ctx != nil {
		context.AfterFunc(ctx, func() { _ = out.Close() })
	}
	return out, nil
}

type win struct {
	*window.Buffer
	mu  sync.Mutex
	wnd objc.ID
}

func (w *win) create(title string, width, height int) error {
	rect := nsRect{Size: nsSize{Width: float64(width), Height: float64(height)}}
	wnd := objc.ID(objc.GetClass("NSWindow")).Send(objc.RegisterName("alloc"))
	wnd = wnd.Send(objc.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect,
		nsWindowStyleTitled|nsWindowStyleClosable|nsWindowStyleMiniaturizable|nsWindowStyleResizable,
		nsBackingBuffered,
		false,
	)
	if wnd == 0 {
		return fmt.Errorf("NSWindow init")
	}
	if title != "" {
		ns := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), title)
		wnd.Send(selSetTitle, ns)
	}
	view := wnd.Send(selContentView)
	view.Send(selSetWantsLayer, true)
	wnd.Send(selMakeKeyAndOrderFront, objc.ID(0))
	objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication")).Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
	w.wnd = wnd
	return nil
}

func (w *win) Draw() error {
	if err := w.Swap(); err != nil {
		return err
	}
	thread.Go(w.flush)
	return nil
}

func (w *win) flush() {
	if w.Closed() {
		return
	}
	if ww, hh := w.clientSize(); ww > 0 && hh > 0 {
		_ = w.Buffer.Resize(image.Pt(ww, hh))
	}
	_ = w.blit()
}

func (w *win) Resize(size image.Point) error {
	if err := w.Buffer.Resize(size); err != nil {
		return err
	}
	onApp(func() {
		w.mu.Lock()
		wnd := w.wnd
		w.mu.Unlock()
		if wnd == 0 {
			return
		}
		s := w.scale()
		if s < 1 {
			s = 1
		}
		wnd.Send(selSetContentSize, nsSize{Width: float64(size.X) / s, Height: float64(size.Y) / s})
	})
	return nil
}

func (w *win) Close() error {
	live.Delete(w)
	thread.Go(func() { w.closeNS() })
	return w.Buffer.Close()
}

func (w *win) closeNS() {
	w.mu.Lock()
	wnd := w.wnd
	w.wnd = 0
	w.mu.Unlock()
	if wnd != 0 {
		wnd.Send(selClose)
	}
}

// note runs on the AppKit thread.
func (w *win) note() {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return
	}
	if wnd.Send(selIsVisible) == 0 {
		_ = w.Close()
		return
	}
	width, height := w.clientSize()
	if width > 0 && height > 0 {
		_ = w.Buffer.Resize(image.Pt(width, height))
	}
	_ = w.blit()
}

func (w *win) clientSize() (int, int) {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return 0, 0
	}
	view := wnd.Send(selContentView)
	if view == 0 {
		return 0, 0
	}
	r := boundsOf(view)
	s := w.scale()
	if s < 1 {
		s = 1
	}
	return int(r.Size.Width*s + 0.5), int(r.Size.Height*s + 0.5)
}

func (w *win) scale() float64 {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return 1
	}
	s := backingScale(wnd)
	if s < 1 {
		return 1
	}
	return s
}

func (w *win) blit() error {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return window.ErrClosed
	}
	var err error
	w.WithFront(func(src *image.RGBA) {
		if src.Rect.Dx() < 1 || src.Rect.Dy() < 1 {
			return
		}
		var cg uintptr
		cg, err = cgImageFromRGBA(src)
		if err != nil {
			return
		}
		defer cgImageRelease(cg)
		view := wnd.Send(selContentView)
		layer := view.Send(selLayer)
		prepareLayer(layer, w.scale())
		beginNoAnim()
		layer.Send(selSetContents, objc.ID(cg))
		endNoAnim()
	})
	return err
}

func beginNoAnim() {
	tx := objc.ID(objc.GetClass("CATransaction"))
	tx.Send(objc.RegisterName("begin"))
	setBool(tx, objc.RegisterName("setDisableActions:"), true)
}

func endNoAnim() {
	objc.ID(objc.GetClass("CATransaction")).Send(objc.RegisterName("commit"))
}

func prepareLayer(layer objc.ID, scale float64) {
	setLayerScale(layer, scale)
	nearest := nsstr("nearest")
	setID(layer, selSetMagFilter, nearest)
	setID(layer, selSetMinFilter, nearest)
	setID(layer, selSetContentsGravity, nsstr("resize"))
	setBool(layer, selSetAllowsEdgeAntialiasing, false)
	setMask(layer, selSetEdgeAntialiasingMask, 0)
}

const (
	cgImageAlphaLast         = 3
	cgBitmapByteOrder32Big   = 4 << 12
	cgRenderingIntentDefault = 0
)

var (
	cgColorSpaceCreateDeviceRGB    func() uintptr
	cgDataProviderCreateWithCFData func(uintptr) uintptr
	cgImageCreate                  func(width, height, bpc, bpp, bpr uintptr, space uintptr, bitmapInfo uint32, provider, decode uintptr, interpolate bool, intent int32) uintptr
	cgColorSpaceRelease            func(uintptr)
	cgDataProviderRelease          func(uintptr)
	cgImageRelease                 func(uintptr)
)

func loadCG() {
	cg, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_LAZY)
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&cgColorSpaceCreateDeviceRGB, cg, "CGColorSpaceCreateDeviceRGB")
	purego.RegisterLibFunc(&cgDataProviderCreateWithCFData, cg, "CGDataProviderCreateWithCFData")
	purego.RegisterLibFunc(&cgImageCreate, cg, "CGImageCreate")
	purego.RegisterLibFunc(&cgColorSpaceRelease, cg, "CGColorSpaceRelease")
	purego.RegisterLibFunc(&cgDataProviderRelease, cg, "CGDataProviderRelease")
	purego.RegisterLibFunc(&cgImageRelease, cg, "CGImageRelease")
}

func cgImageFromRGBA(src *image.RGBA) (uintptr, error) {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	if w < 1 || h < 1 || cgImageCreate == nil {
		return 0, fmt.Errorf("cgimage")
	}
	data := objc.ID(objc.GetClass("NSData")).Send(
		objc.RegisterName("dataWithBytes:length:"),
		uintptr(unsafe.Pointer(&src.Pix[0])),
		len(src.Pix),
	)
	if data == 0 {
		return 0, fmt.Errorf("NSData")
	}
	space := cgColorSpaceCreateDeviceRGB()
	if space == 0 {
		return 0, fmt.Errorf("CGColorSpaceCreateDeviceRGB")
	}
	defer cgColorSpaceRelease(space)
	provider := cgDataProviderCreateWithCFData(uintptr(data))
	if provider == 0 {
		return 0, fmt.Errorf("CGDataProviderCreateWithCFData")
	}
	defer cgDataProviderRelease(provider)
	img := cgImageCreate(
		uintptr(w), uintptr(h), 8, 32, uintptr(src.Stride),
		space,
		cgImageAlphaLast|cgBitmapByteOrder32Big,
		provider, 0,
		false,
		cgRenderingIntentDefault,
	)
	if img == 0 {
		return 0, fmt.Errorf("CGImageCreate")
	}
	return img, nil
}

func nsstr(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), s)
}

var (
	boundsFn  func(objc.ID, objc.SEL) nsRect
	scaleFn   func(objc.ID, objc.SEL) float64
	setScale  func(objc.ID, objc.SEL, float64)
	setIDFn   func(objc.ID, objc.SEL, objc.ID)
	setBoolFn func(objc.ID, objc.SEL, bool)
	setMaskFn func(objc.ID, objc.SEL, uint32)
)

func setID(obj objc.ID, sel objc.SEL, v objc.ID) {
	if setIDFn == nil {
		purego.RegisterFunc(&setIDFn, objcMsgSend)
	}
	setIDFn(obj, sel, v)
}

func setBool(obj objc.ID, sel objc.SEL, v bool) {
	if setBoolFn == nil {
		purego.RegisterFunc(&setBoolFn, objcMsgSend)
	}
	setBoolFn(obj, sel, v)
}

func setMask(obj objc.ID, sel objc.SEL, v uint32) {
	if setMaskFn == nil {
		purego.RegisterFunc(&setMaskFn, objcMsgSend)
	}
	setMaskFn(obj, sel, v)
}

func boundsOf(view objc.ID) nsRect {
	if boundsFn == nil {
		purego.RegisterFunc(&boundsFn, objcMsgSend)
	}
	return boundsFn(view, selBounds)
}

func backingScale(wnd objc.ID) float64 {
	if scaleFn == nil {
		purego.RegisterFunc(&scaleFn, objcMsgSend)
	}
	return scaleFn(wnd, selBackingScaleFactor)
}

func setLayerScale(layer objc.ID, scale float64) {
	if setScale == nil {
		purego.RegisterFunc(&setScale, objcMsgSend)
	}
	setScale(layer, selSetContentsScale, scale)
}

var objcMsgSend = mustSym("/usr/lib/libobjc.A.dylib", "objc_msgSend")

func mustSym(lib, name string) uintptr {
	h, err := purego.Dlopen(lib, purego.RTLD_LAZY)
	if err != nil {
		panic(err)
	}
	sym, err := purego.Dlsym(h, name)
	if err != nil {
		panic(err)
	}
	return sym
}
