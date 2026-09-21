//go:build darwin

package cocoa

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi"
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
	selDataNoCopy                = objc.RegisterName("dataWithBytesNoCopy:length:freeWhenDone:")
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
	selInLiveResize              = objc.RegisterName("inLiveResize")
	selBounds                    = objc.RegisterName("bounds")
	selBackingScaleFactor        = objc.RegisterName("backingScaleFactor")
	selScreen                    = objc.RegisterName("screen")
	selMainScreen                = objc.RegisterName("mainScreen")
	selMaximumFramesPerSecond    = objc.RegisterName("maximumFramesPerSecond")

	live             sync.Map // *win → struct{}
	selNextEvent     = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent     = objc.RegisterName("sendEvent:")
	selUpdateWindows = objc.RegisterName("updateWindows")
	selDistantPast   = objc.RegisterName("distantPast")
	selLockSurface   = objc.RegisterName("lockWithOptions:seed:")
	selUnlockSurface = objc.RegisterName("unlockWithOptions:seed:")
	selBaseAddress   = objc.RegisterName("baseAddress")
	selBytesPerRow   = objc.RegisterName("bytesPerRow")
	selRelease       = objc.RegisterName("release")
	selSetObjectKey  = objc.RegisterName("setObject:forKey:")
	selNumberInteger = objc.RegisterName("numberWithInteger:")
	selDictionary    = objc.RegisterName("dictionary")
	selAlloc         = objc.RegisterName("alloc")
	selInitProps     = objc.RegisterName("initWithProperties:")

	runLoopMode objc.ID
	colorSpace  uintptr
)

const pixelFormatRGBA = 0x52474241 // 'RGBA'

func startApp() error {
	if !thread.Bound() {
		return fmt.Errorf("%w", window.ErrNotBound)
	}
	appOnce.Do(func() {
		thread.Do(func() {
			if !thread.ProcessMain() {
				appErr = fmt.Errorf("%w", window.ErrNotMain)
				return
			}
			if _, err := ffi.Open("/System/Library/Frameworks/Cocoa.framework/Cocoa", ffi.Global|ffi.Lazy); err != nil {
				appErr = fmt.Errorf("%w: cocoa: %w", window.ErrInit, err)
				return
			}
			if err := loadCG(); err != nil {
				appErr = fmt.Errorf("%w: coregraphics: %w", window.ErrInit, err)
				return
			}
			if _, err := ffi.Open("/System/Library/Frameworks/IOSurface.framework/IOSurface", ffi.Global|ffi.Lazy); err != nil {
				slog.Debug("cocoa IOSurface missing", "err", err)
			}
			runLoopMode = nsstr("kCFRunLoopDefaultMode").Send(objc.RegisterName("retain"))
			colorSpace = cgColorSpaceCreateDeviceRGB()
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

func withPool(fn func()) {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(objc.RegisterName("new"))
	defer pool.Send(objc.RegisterName("drain"))
	fn()
}

func pump(app objc.ID) {
	if !thread.ProcessMain() {
		return
	}
	withPool(func() {
		pumpInner(app)
	})
}

func pumpInner(app objc.ID) {
	date := objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
	for {
		ev := app.Send(selNextEvent, ^uintptr(0), date, runLoopMode, true)
		if ev == 0 {
			break
		}
		dispatchInput(ev)
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
			if width, height := out.clientSize(); width > 0 && height > 0 {
				out.wantWidth, out.wantHeight = width, height
				_ = out.Buffer.Resize(image.Pt(width, height))
			}
			period := cfg.Period
			if period <= 0 {
				period = screenFramePeriod(out.wnd)
			}
			out.SetFramePeriod(period)
		}
	})
	if openErr != nil {
		return nil, openErr
	}
	live.Store(out, struct{}{})
	window.CloseWhenDone(ctx, out)
	return out, nil
}

type win struct {
	*window.Buffer
	mu             sync.Mutex
	paint          sync.Mutex
	dead           bool
	wnd            objc.ID
	pixelCopy      [2][]byte
	pixelCopyIndex int
	layerScale     float64
	surfaces       [2]objc.ID
	surfaceIndex   int
	surfaceWidth   int
	surfaceHeight  int
	surfaceStride  int
	stale          []objc.ID
	wantWidth      int
	wantHeight     int
	liveResize     bool
	pending        objc.ID
	displayed      objc.ID
	pendingBlit    bool
	mapped         objc.ID
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
		return fmt.Errorf("%w", window.ErrInit)
	}
	if title != "" {
		ns := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), title)
		wnd.Send(selSetTitle, ns)
	}
	view := wnd.Send(selContentView)
	view.Send(selSetWantsLayer, true)
	acceptMouseMoved(wnd)
	wnd.Send(selMakeKeyAndOrderFront, objc.ID(0))
	objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication")).Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
	w.wnd = wnd
	return nil
}

func (w *win) Size() image.Point {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wantWidth > 0 && w.wantHeight > 0 {
		return image.Pt(w.wantWidth, w.wantHeight)
	}
	return w.Buffer.Size()
}

func (w *win) Frame() *image.RGBA {
	w.paint.Lock()
	w.mu.Lock()
	dead := w.dead
	w.mu.Unlock()
	if dead {
		w.paint.Unlock()
		return nil
	}
	want := w.Size()
	if want.X > 0 && want.Y > 0 {
		_, _ = w.Buffer.EnsureSize(want)
	}
	w.unlockMapped()
	width, height := want.X, want.Y
	if width < 1 || height < 1 {
		return w.Buffer.Frame()
	}
	stride := width * 4
	w.mu.Lock()
	ok := w.ensureSurfaces(width, height, stride)
	surface := w.surfaces[w.surfaceIndex]
	w.mu.Unlock()
	if !ok || surface == 0 {
		return w.Buffer.Frame()
	}
	rgba := lockRGBA(surface, width, height)
	if rgba == nil {
		return w.Buffer.Frame()
	}
	w.mapped = surface
	return rgba
}

func (w *win) unlockMapped() {
	if w.mapped == 0 {
		return
	}
	w.mapped.Send(selUnlockSurface, 0, 0)
	w.mapped = 0
}

func (w *win) Draw() error {
	defer w.paint.Unlock()
	w.mu.Lock()
	dead := w.dead
	w.mu.Unlock()
	if dead {
		w.unlockMapped()
		return nil
	}
	if w.mapped != 0 {
		surface := w.mapped
		w.unlockMapped()
		w.mu.Lock()
		w.surfaceIndex ^= 1
		w.mu.Unlock()
		w.queuePresent(surface, false)
		return nil
	}
	if err := w.Swap(); err != nil {
		return err
	}
	front := w.Front()
	if front == nil {
		return nil
	}
	width, height, stride := front.Rect.Dx(), front.Rect.Dy(), front.Stride
	if width <= 0 || height <= 0 {
		return nil
	}
	w.mu.Lock()
	ok := w.ensureSurfaces(width, height, stride)
	surface := w.surfaces[w.surfaceIndex]
	if ok {
		w.surfaceIndex ^= 1
	}
	w.mu.Unlock()
	if !ok || surface == 0 {
		w.queuePresent(0, true)
		return nil
	}
	copied := false
	w.WithFront(func(src *image.RGBA) {
		if src.Rect.Dx() != width || src.Rect.Dy() != height || src.Stride != stride {
			return
		}
		copied = copyIOSurface(surface, src)
	})
	if !copied {
		w.queuePresent(0, true)
		return nil
	}
	w.queuePresent(surface, false)
	return nil
}

func (w *win) queuePresent(surface objc.ID, blit bool) {
	w.mu.Lock()
	if blit {
		w.pendingBlit = true
		w.pending = 0
	} else {
		w.pending = surface
		w.pendingBlit = false
	}
	w.mu.Unlock()
	thread.Enqueue(w.flushPending)
}

func (w *win) flushPending() {
	for {
		w.mu.Lock()
		surface := w.pending
		blit := w.pendingBlit
		w.pending = 0
		w.pendingBlit = false
		w.mu.Unlock()
		if surface == 0 && !blit {
			return
		}
		if blit {
			_ = w.blit()
			continue
		}
		w.setContents(surface)
	}
}

func (w *win) Resize(size image.Point) error {
	w.mu.Lock()
	w.wantWidth, w.wantHeight = size.X, size.Y
	w.mu.Unlock()
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
		scale := w.scale()
		if scale < 1 {
			scale = 1
		}
		wnd.Send(selSetContentSize, nsSize{Width: float64(size.X) / scale, Height: float64(size.Y) / scale})
	})
	return nil
}

func (w *win) Close() error {
	live.Delete(w)
	w.paint.Lock()
	w.mu.Lock()
	w.dead = true
	w.mu.Unlock()
	w.unlockMapped()
	w.paint.Unlock()
	thread.Go(func() { w.closeNS() })
	return w.Buffer.Close()
}

func (w *win) closeNS() {
	w.paint.Lock()
	defer w.paint.Unlock()
	w.mu.Lock()
	wnd := w.wnd
	surfaces := w.surfaces
	stale := w.stale
	w.wnd = 0
	w.surfaces = [2]objc.ID{}
	w.stale = nil
	w.pending = 0
	w.displayed = 0
	w.mu.Unlock()
	for _, surface := range surfaces {
		if surface != 0 {
			surface.Send(selRelease)
		}
	}
	for _, surface := range stale {
		if surface != 0 {
			surface.Send(selRelease)
		}
	}
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
	view := wnd.Send(selContentView)
	live := view != 0 && view.Send(selInLiveResize) != 0
	if wnd.Send(selIsVisible) == 0 {
		if live {
			return
		}
		_ = w.Close()
		return
	}
	width, height := w.clientSize()
	if width < 1 || height < 1 {
		return
	}
	w.mu.Lock()
	wasLive := w.liveResize
	w.liveResize = live
	changed := w.wantWidth != width || w.wantHeight != height
	w.wantWidth, w.wantHeight = width, height
	w.mu.Unlock()
	if wasLive && !live {
		w.Emit(window.Pointer{Button: 1, Pressed: false})
	}
	if changed {
		w.Emit(window.Resize{Size: image.Pt(width, height)})
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
	rect := boundsOf(view)
	scale := w.scale()
	if scale < 1 {
		scale = 1
	}
	return int(rect.Size.Width*scale + 0.5), int(rect.Size.Height*scale + 0.5)
}

func (w *win) scale() float64 {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return 1
	}
	scale := backingScale(wnd)
	if scale < 1 {
		return 1
	}
	return scale
}

func (w *win) setContents(surface objc.ID) {
	if surface == 0 {
		return
	}
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return
	}
	withPool(func() {
		view := wnd.Send(selContentView)
		layer := view.Send(selLayer)
		scale := w.scale()
		if w.layerScale != scale {
			prepareLayer(layer, scale)
			w.layerScale = scale
		}
		layer.Send(selSetContents, surface)
	})
	w.mu.Lock()
	w.displayed = surface
	w.mu.Unlock()
	w.releaseStale(surface)
}

func (w *win) retire(surface objc.ID) {
	if surface != 0 {
		w.stale = append(w.stale, surface)
	}
}

func (w *win) releaseStale(keep objc.ID) {
	w.mu.Lock()
	stale := w.stale
	w.stale = nil
	live0, live1 := w.surfaces[0], w.surfaces[1]
	pending, displayed := w.pending, w.displayed
	w.mu.Unlock()
	var hold []objc.ID
	for _, surface := range stale {
		if surface == 0 || surface == live0 || surface == live1 {
			continue
		}
		if surface == pending || surface == displayed || surface == keep {
			hold = append(hold, surface)
			continue
		}
		surface.Send(selRelease)
	}
	if len(hold) == 0 {
		return
	}
	w.mu.Lock()
	w.stale = append(w.stale, hold...)
	w.mu.Unlock()
}

func (w *win) blit() error {
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return window.ErrClosed
	}
	var err error
	withPool(func() {
		w.WithFront(func(src *image.RGBA) {
			if src.Rect.Dx() < 1 || src.Rect.Dy() < 1 {
				return
			}
			view := wnd.Send(selContentView)
			layer := view.Send(selLayer)
			scale := w.scale()
			if w.layerScale != scale {
				prepareLayer(layer, scale)
				w.layerScale = scale
			}
			var cgImage uintptr
			cgImage, err = w.cgImageFromRGBA(src)
			if err != nil {
				return
			}
			layer.Send(selSetContents, objc.ID(cgImage))
			cgImageRelease(cgImage)
		})
	})
	return err
}

var (
	filterNearest objc.ID
	gravityResize objc.ID
	filterOnce    sync.Once
)

func prepareLayer(layer objc.ID, scale float64) {
	filterOnce.Do(func() {
		retain := objc.RegisterName("retain")
		filterNearest = nsstr("nearest").Send(retain)
		gravityResize = nsstr("resize").Send(retain)
	})
	setLayerScale(layer, scale)
	setID(layer, selSetMagFilter, filterNearest)
	setID(layer, selSetMinFilter, filterNearest)
	setID(layer, selSetContentsGravity, gravityResize)
	setBool(layer, selSetAllowsEdgeAntialiasing, false)
	setMask(layer, selSetEdgeAntialiasingMask, 0)
	null := objc.ID(objc.GetClass("NSNull")).Send(objc.RegisterName("null"))
	actions := objc.ID(objc.GetClass("NSMutableDictionary")).Send(selDictionary)
	actions.Send(selSetObjectKey, null, nsstr("contents"))
	layer.Send(objc.RegisterName("setActions:"), actions)
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

func loadCG() error {
	lib, err := ffi.Open("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", ffi.Lazy)
	if err != nil {
		return err
	}
	ffi.Func(lib, "CGColorSpaceCreateDeviceRGB", &cgColorSpaceCreateDeviceRGB)
	ffi.Func(lib, "CGDataProviderCreateWithCFData", &cgDataProviderCreateWithCFData)
	ffi.Func(lib, "CGImageCreate", &cgImageCreate)
	ffi.Func(lib, "CGColorSpaceRelease", &cgColorSpaceRelease)
	ffi.Func(lib, "CGDataProviderRelease", &cgDataProviderRelease)
	ffi.Func(lib, "CGImageRelease", &cgImageRelease)
	return nil
}

func nsNumber(value int) objc.ID {
	return objc.ID(objc.GetClass("NSNumber")).Send(selNumberInteger, value)
}

func (w *win) ensureSurfaces(width, height, stride int) bool {
	if w.surfaces[0] != 0 && w.surfaces[1] != 0 && w.surfaceWidth == width && w.surfaceHeight == height && w.surfaceStride == stride {
		return true
	}
	class := objc.GetClass("IOSurface")
	if class == 0 {
		return false
	}
	var created [2]objc.ID
	ok := true
	withPool(func() {
		for i := range created {
			created[i] = newIOSurface(class, width, height, stride)
			if created[i] == 0 {
				ok = false
				return
			}
		}
	})
	if !ok {
		for _, surface := range created {
			if surface != 0 {
				surface.Send(selRelease)
			}
		}
		return false
	}
	w.retire(w.surfaces[0])
	w.retire(w.surfaces[1])
	w.surfaces = created
	w.surfaceWidth = width
	w.surfaceHeight = height
	w.surfaceStride = stride
	w.surfaceIndex = 0
	slog.Debug("cocoa IOSurface", "width", width, "height", height, "stride", stride)
	return true
}

const ioSurfaceAlign = 64

func alignedBytes(n int) int {
	if n < 1 {
		return ioSurfaceAlign
	}
	return (n + ioSurfaceAlign - 1) &^ (ioSurfaceAlign - 1)
}

func newIOSurface(class objc.Class, width, height, stride int) objc.ID {
	row := alignedBytes(stride)
	properties := objc.ID(objc.GetClass("NSMutableDictionary")).Send(selDictionary)
	properties.Send(selSetObjectKey, nsNumber(width), nsstr("IOSurfaceWidth"))
	properties.Send(selSetObjectKey, nsNumber(height), nsstr("IOSurfaceHeight"))
	properties.Send(selSetObjectKey, nsNumber(4), nsstr("IOSurfaceBytesPerElement"))
	properties.Send(selSetObjectKey, nsNumber(row), nsstr("IOSurfaceBytesPerRow"))
	properties.Send(selSetObjectKey, nsNumber(row*height), nsstr("IOSurfaceAllocSize"))
	properties.Send(selSetObjectKey, nsNumber(pixelFormatRGBA), nsstr("IOSurfacePixelFormat"))
	return objc.ID(class).Send(selAlloc).Send(selInitProps, properties)
}

func lockRGBA(surface objc.ID, width, height int) *image.RGBA {
	if surface == 0 || width < 1 || height < 1 {
		return nil
	}
	if surface.Send(selLockSurface, 0, 0) != 0 {
		return nil
	}
	base := surface.Send(selBaseAddress)
	rowBytes := int(surface.Send(selBytesPerRow))
	row := width * 4
	if base == 0 || rowBytes < row {
		surface.Send(selUnlockSurface, 0, 0)
		return nil
	}
	pix := unsafe.Slice((*byte)(unsafe.Pointer(base)), rowBytes*height)
	return &image.RGBA{Pix: pix, Stride: rowBytes, Rect: image.Rect(0, 0, width, height)}
}

func copyIOSurface(surface objc.ID, source *image.RGBA) bool {
	if surface == 0 || source == nil {
		return false
	}
	width, height := source.Rect.Dx(), source.Rect.Dy()
	stride := source.Stride
	row := width * 4
	if width < 1 || height < 1 || stride < row || len(source.Pix) < (height-1)*stride+row {
		return false
	}
	if surface.Send(selLockSurface, 0, 0) != 0 {
		return false
	}
	base := surface.Send(selBaseAddress)
	if base == 0 {
		surface.Send(selUnlockSurface, 0, 0)
		return false
	}
	rowBytes := int(surface.Send(selBytesPerRow))
	if rowBytes < row {
		surface.Send(selUnlockSurface, 0, 0)
		return false
	}
	destination := unsafe.Slice((*byte)(unsafe.Pointer(base)), rowBytes*height)
	if rowBytes == stride {
		copy(destination[:height*stride], source.Pix[:height*stride])
	} else {
		for y := 0; y < height; y++ {
			copy(destination[y*rowBytes:y*rowBytes+row], source.Pix[y*stride:y*stride+row])
		}
	}
	surface.Send(selUnlockSurface, 0, 0)
	return true
}

func (w *win) cgImageFromRGBA(src *image.RGBA) (uintptr, error) {
	width, height := src.Rect.Dx(), src.Rect.Dy()
	if width < 1 || height < 1 || cgImageCreate == nil || len(src.Pix) == 0 {
		return 0, fmt.Errorf("%w: empty", window.ErrPresent)
	}
	n := len(src.Pix)
	i := w.pixelCopyIndex
	w.pixelCopyIndex ^= 1
	buf := &w.pixelCopy[i]
	if cap(*buf) < n {
		*buf = make([]byte, n)
	} else {
		*buf = (*buf)[:n]
	}
	copy(*buf, src.Pix)
	data := objc.ID(objc.GetClass("NSData")).Send(
		selDataNoCopy,
		uintptr(unsafe.Pointer(&(*buf)[0])),
		uintptr(n),
		false,
	)
	if data == 0 {
		return 0, fmt.Errorf("%w: NSData", window.ErrPresent)
	}
	if colorSpace == 0 {
		return 0, fmt.Errorf("%w: color space", window.ErrPresent)
	}
	provider := cgDataProviderCreateWithCFData(uintptr(data))
	if provider == 0 {
		return 0, fmt.Errorf("%w: data provider", window.ErrPresent)
	}
	defer cgDataProviderRelease(provider)
	img := cgImageCreate(
		uintptr(width), uintptr(height), 8, 32, uintptr(src.Stride),
		colorSpace,
		cgImageAlphaLast|cgBitmapByteOrder32Big,
		provider, 0,
		false,
		cgRenderingIntentDefault,
	)
	if img == 0 {
		return 0, fmt.Errorf("%w: CGImage", window.ErrPresent)
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
		ffi.Register(&setIDFn, objcMsgSend)
	}
	setIDFn(obj, sel, v)
}

func setBool(obj objc.ID, sel objc.SEL, v bool) {
	if setBoolFn == nil {
		ffi.Register(&setBoolFn, objcMsgSend)
	}
	setBoolFn(obj, sel, v)
}

func setMask(obj objc.ID, sel objc.SEL, v uint32) {
	if setMaskFn == nil {
		ffi.Register(&setMaskFn, objcMsgSend)
	}
	setMaskFn(obj, sel, v)
}

func boundsOf(view objc.ID) nsRect {
	if boundsFn == nil {
		ffi.Register(&boundsFn, objcMsgSend)
	}
	return boundsFn(view, selBounds)
}

func screenFramePeriod(wnd objc.ID) time.Duration {
	screen := objc.ID(0)
	if wnd != 0 {
		screen = wnd.Send(selScreen)
	}
	if screen == 0 {
		screen = objc.ID(objc.GetClass("NSScreen")).Send(selMainScreen)
	}
	if screen == 0 {
		return 0
	}
	fps := int(screen.Send(selMaximumFramesPerSecond))
	if fps <= 0 {
		return 0
	}
	return time.Second / time.Duration(fps)
}

func backingScale(wnd objc.ID) float64 {
	if scaleFn == nil {
		ffi.Register(&scaleFn, objcMsgSend)
	}
	return scaleFn(wnd, selBackingScaleFactor)
}

func setLayerScale(layer objc.ID, scale float64) {
	if setScale == nil {
		ffi.Register(&setScale, objcMsgSend)
	}
	setScale(layer, selSetContentsScale, scale)
}

var objcMsgSend = mustSymbol("/usr/lib/libobjc.A.dylib", "objc_msgSend")

func mustSymbol(lib, name string) uintptr {
	handle, err := ffi.Open(lib, ffi.Lazy)
	if err != nil {
		panic(err)
	}
	sym, err := ffi.Symbol(handle, name)
	if err != nil {
		panic(err)
	}
	return sym
}
