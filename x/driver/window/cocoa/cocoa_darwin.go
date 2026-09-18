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

	selContentView          = objc.RegisterName("contentView")
	selSetWantsLayer        = objc.RegisterName("setWantsLayer:")
	selLayer                = objc.RegisterName("layer")
	selSetContents          = objc.RegisterName("setContents:")
	selSetTitle             = objc.RegisterName("setTitle:")
	selMakeKeyAndOrderFront = objc.RegisterName("makeKeyAndOrderFront:")
	selSetContentSize       = objc.RegisterName("setContentSize:")
	selClose                = objc.RegisterName("close")
	selIsVisible            = objc.RegisterName("isVisible")
	selBounds               = objc.RegisterName("bounds")

	live             sync.Map // *win → struct{}
	selNextEvent     = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent     = objc.RegisterName("sendEvent:")
	selUpdateWindows = objc.RegisterName("updateWindows")
	selBitmapData    = objc.RegisterName("bitmapData")
	selDistantPast   = objc.RegisterName("distantPast")
)

func startApp() error {
	if !thread.Bound() {
		return fmt.Errorf("cocoa: thread.Bind was not called from main")
	}
	appOnce.Do(func() {
		thread.Do(func() {
			if _, err := purego.Dlopen("/System/Library/Frameworks/Cocoa.framework/Cocoa", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
				appErr = err
				return
			}
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
	mu       sync.Mutex
	wnd      objc.ID
	rep, img objc.ID
	bw, bh   int
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
	if w.Closed() {
		return window.ErrClosed
	}
	var err error
	onApp(func() {
		if ww, hh := w.clientSize(); ww > 0 && hh > 0 {
			_ = w.Buffer.Resize(image.Pt(ww, hh))
		}
		err = w.blit()
	})
	return err
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
		wnd.Send(selSetContentSize, nsSize{Width: float64(size.X), Height: float64(size.Y)})
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
	return int(r.Size.Width + 0.5), int(r.Size.Height + 0.5)
}

func (w *win) blit() error {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return window.ErrClosed
	}
	src := w.Frame()
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if sw < 1 || sh < 1 {
		return nil
	}
	if w.rep == 0 || w.bw != sw || w.bh != sh {
		if err := w.makeRep(sw, sh, src.Stride); err != nil {
			return err
		}
	}
	dst := w.rep.Send(selBitmapData)
	if dst == 0 {
		return fmt.Errorf("NSBitmapImageRep bitmapData")
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(dst)), len(src.Pix)), src.Pix)
	view := wnd.Send(selContentView)
	layer := view.Send(selLayer)
	layer.Send(selSetContents, w.img)
	return nil
}

func (w *win) makeRep(width, height, stride int) error {
	rep := objc.ID(objc.GetClass("NSBitmapImageRep")).Send(objc.RegisterName("alloc"))
	cs := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), "NSCalibratedRGBColorSpace")
	rep = rep.Send(objc.RegisterName("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:"),
		uintptr(0), width, height, 8, 4, true, false, cs, stride, 32,
	)
	if rep == 0 {
		return fmt.Errorf("NSBitmapImageRep init")
	}
	img := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc"))
	img = img.Send(objc.RegisterName("initWithSize:"), nsSize{Width: float64(width), Height: float64(height)})
	img.Send(objc.RegisterName("addRepresentation:"), rep)
	w.rep, w.img = rep, img
	w.bw, w.bh = width, height
	return nil
}

var boundsFn func(objc.ID, objc.SEL) nsRect

func boundsOf(view objc.ID) nsRect {
	if boundsFn == nil {
		purego.RegisterFunc(&boundsFn, objcMsgSend)
	}
	return boundsFn(view, selBounds)
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
