//go:build darwin

package metal

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	instanceStride = 64
	kindView       = 3
	kindUIView     = 5
)

var (
	// ErrClosed means Draw or Adopt ran after Close.
	ErrClosed = errors.New("metal closed")
	// ErrLost means the view or the drawable is gone.
	ErrLost = errors.New("metal surface lost")
	// ErrSize means the frame size or instance bytes are not usable.
	ErrSize = errors.New("metal size")
	// ErrUnavailable means Metal or the shader failed to open.
	ErrUnavailable = errors.New("metal unavailable")
)

// Screen is one Metal drawable attached to a view the caller owns.
type Screen struct {
	mu       sync.Mutex
	device   objc.ID
	queue    objc.ID
	lib      objc.ID
	layer    objc.ID
	view     objc.ID
	clear    objc.ID
	fill     objc.ID
	ink      objc.ID
	fillBuf  objc.ID
	underBuf objc.ID
	inkBuf   objc.ID
	fillCap  int
	underCap int
	inkCap   int
	width    int
	height   int
	inkW     int
	inkH     int
	inkReady bool
	closed   bool
}

var (
	uiMu   sync.Mutex
	uiHook func(func())
)

// SetUI runs UI work on the platform thread the driver owns.
// iOS passes the UIKit main queue. macOS passes the process main thread.
func SetUI(fn func(func())) {
	uiMu.Lock()
	uiHook = fn
	uiMu.Unlock()
}

func uiDo(fn func()) {
	uiMu.Lock()
	hook := uiHook
	uiMu.Unlock()
	if hook != nil {
		hook(fn)
		return
	}
	fn()
}

var (
	devOnce  sync.Once
	devErr   error
	devID    objc.ID
	createFn func() uintptr
)

// Available reports whether this process can open a Metal device.
func Available() error {
	_, err := systemDevice()
	return err
}

func systemDevice() (objc.ID, error) {
	devOnce.Do(func() {
		if err := loadFrameworks(); err != nil {
			devErr = err
			return
		}
		if createFn == nil {
			devErr = ErrUnavailable
			return
		}
		id := objc.ID(createFn())
		if id == 0 {
			devErr = ErrUnavailable
			return
		}
		devID = id.Send(selRetain)
	})
	if devErr != nil {
		return 0, devErr
	}
	return devID, nil
}

func loadFrameworks() error {
	for _, path := range []string{
		"/System/Library/Frameworks/Foundation.framework/Foundation",
		"/System/Library/Frameworks/QuartzCore.framework/QuartzCore",
		"/System/Library/Frameworks/Metal.framework/Metal",
	} {
		lib, err := native.Open(path, native.Global|native.Lazy)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrUnavailable, path)
		}
		if path[len(path)-5:] == "Metal" {
			native.Func(lib, "MTLCreateSystemDefaultDevice", &createFn)
		}
	}
	if createFn == nil {
		return ErrUnavailable
	}
	return nil
}

// OpenNative attaches a CAMetalLayer to view and compiles the frame shaders.
// kind is window.SurfaceView (3) or window.SurfaceUIView (5).
// The view is not closed with the screen.
func OpenNative(kind int, view uintptr, width, height int) (*Screen, error) {
	if kind != kindView && kind != kindUIView {
		return nil, ErrUnavailable
	}
	if view == 0 || width < 1 || height < 1 {
		return nil, ErrSize
	}
	device, err := systemDevice()
	if err != nil {
		return nil, err
	}
	screen := &Screen{device: device.Send(selRetain), width: width, height: height}
	var openErr error
	uiDo(func() {
		withPool(func() {
			openErr = screen.open(kind, objc.ID(view), width, height)
		})
	})
	if openErr != nil {
		_ = screen.Close()
		return nil, openErr
	}
	return screen, nil
}

func (s *Screen) open(kind int, view objc.ID, width, height int) error {
	queue := s.device.Send(selNewCommandQueue)
	if queue == 0 {
		return ErrUnavailable
	}
	s.queue = queue
	lib, err := compileLibrary(s.device)
	if err != nil {
		return err
	}
	s.lib = lib
	clear, err := s.pipeline("clear_vert", "clear_frag", false)
	if err != nil {
		return err
	}
	fill, err := s.pipeline("fill_vert", "fill_frag", true)
	if err != nil {
		return err
	}
	ink, err := s.pipeline("ink_vert", "ink_frag", true)
	if err != nil {
		return err
	}
	s.clear, s.fill, s.ink = clear, fill, ink
	return s.attach(kind, view, width, height)
}

func (s *Screen) attach(kind int, view objc.ID, width, height int) error {
	if view == 0 {
		return ErrLost
	}
	if s.layer == 0 {
		layer := objc.ID(objc.GetClass("CAMetalLayer")).Send(selLayer)
		if layer == 0 {
			layer = objc.ID(objc.GetClass("CAMetalLayer")).Send(selAlloc).Send(selInit)
		}
		if layer == 0 {
			return fmt.Errorf("%w: CAMetalLayer", ErrUnavailable)
		}
		layer.Send(selSetDevice, s.device)
		layer.Send(selSetPixelFormat, uintptr(pixelBGRA))
		layer.Send(selSetFramebufferOnly, true)
		layer.Send(selSetOpaque, true)
		if runtime.GOOS != "ios" {
			layer.Send(selSetDisplaySync, true)
		}
		s.layer = layer.Send(selRetain)
	}
	if kind == kindUIView {
		host := view.Send(selLayer)
		if host == 0 {
			return ErrLost
		}
		s.layer.Send(selRemove)
		host.Send(selAddSublayer, s.layer)
		setRect(s.layer, selSetFrame, boundsOf(view))
	} else {
		view.Send(selSetWantsLayer, true)
		view.Send(selSetLayer, s.layer)
	}
	s.view = view
	s.fit(width, height)
	return nil
}

func (s *Screen) fit(width, height int) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	setSize(s.layer, selSetDrawableSize, cgSize{Width: float64(width), Height: float64(height)})
	s.width, s.height = width, height
}

// Adopt moves the layer onto a replacement view.
func (s *Screen) Adopt(view uintptr, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.device == 0 {
		return ErrClosed
	}
	if view == 0 {
		return ErrLost
	}
	kind := kindView
	if runtime.GOOS == "ios" {
		kind = kindUIView
	}
	var err error
	uiDo(func() {
		withPool(func() {
			if s.view == objc.ID(view) {
				s.fit(width, height)
				if kind == kindUIView && s.view != 0 {
					setRect(s.layer, selSetFrame, boundsOf(s.view))
				}
				return
			}
			err = s.attach(kind, objc.ID(view), width, height)
		})
	})
	return err
}

// Draw paints under, then the rounded rects, then ink, and presents.
// The UI hook only resizes the layer. Encode, present, and the GPU wait
// run on the caller so the main queue can finish the present.
func (s *Screen) Draw(instances, under, ink []byte, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	if width < 1 || height < 1 || len(instances)%instanceStride != 0 {
		return ErrSize
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.device == 0 {
		return ErrClosed
	}
	if err := s.syncFrame(width, height); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var err error
	withPool(func() {
		err = s.draw(instances, under, ink, width, height)
	})
	return err
}

// syncFrame fits the layer on the UI hook. A matching size skips the hook
// so a steady frame does not take the thread that pumps AppKit.
func (s *Screen) syncFrame(width, height int) error {
	if s.layer != 0 && s.view != 0 && width == s.width && height == s.height {
		return nil
	}
	var err error
	uiDo(func() {
		withPool(func() {
			if s.layer == 0 || s.view == 0 {
				err = ErrLost
				return
			}
			if width == s.width && height == s.height {
				return
			}
			s.fit(width, height)
			if runtime.GOOS == "ios" {
				setRect(s.layer, selSetFrame, boundsOf(s.view))
			}
		})
	})
	return err
}

func (s *Screen) draw(instances, under, ink []byte, width, height int) error {
	if s.layer == 0 || s.view == 0 {
		return ErrLost
	}
	need := width * height * 4
	fills := len(instances) / instanceStride
	if fills > 0 {
		if err := s.grow(&s.fillBuf, &s.fillCap, instances); err != nil {
			return err
		}
	}
	backed := len(under) >= need
	if backed {
		if err := s.grow(&s.underBuf, &s.underCap, under[:need]); err != nil {
			return err
		}
	}
	uploadInk := len(ink) >= need
	reuseInk := !uploadInk && s.inkReady && s.inkW == width && s.inkH == height && s.inkBuf != 0
	if uploadInk {
		if err := s.grow(&s.inkBuf, &s.inkCap, ink[:need]); err != nil {
			return err
		}
		s.inkReady = true
		s.inkW, s.inkH = width, height
	}
	inked := uploadInk || reuseInk
	drawable := s.layer.Send(selNextDrawable)
	if drawable == 0 {
		return ErrLost
	}
	cmd := s.queue.Send(selCommandBuffer)
	if cmd == 0 {
		return ErrUnavailable
	}
	pass := objc.ID(objc.GetClass("MTLRenderPassDescriptor")).Send(selRenderPass)
	if pass == 0 {
		return ErrUnavailable
	}
	att := pass.Send(selColorAttachments).Send(selSubscript, uintptr(0))
	att.Send(selSetTexture, drawable.Send(selTexture))
	att.Send(selSetLoadAction, uintptr(loadDontCare))
	att.Send(selSetStoreAction, uintptr(storeStore))
	enc := cmd.Send(selEncoder, pass)
	if enc == 0 {
		return ErrUnavailable
	}
	enc.Send(selSetPipeline, s.clear)
	enc.Send(selDraw, uintptr(primStrip), uintptr(0), uintptr(4), uintptr(1))
	if backed {
		s.blit(enc, s.underBuf, width, height)
	}
	if fills > 0 {
		push := fillPush{extentX: float32(width), extentY: float32(height), swapRB: channelSwap}
		enc.Send(selSetPipeline, s.fill)
		enc.Send(selSetVertexBuffer, s.fillBuf, uintptr(0), uintptr(0))
		enc.Send(selSetFragmentBuffer, s.fillBuf, uintptr(0), uintptr(0))
		enc.Send(selSetVertexBytes, unsafe.Pointer(&push), uintptr(unsafe.Sizeof(push)), uintptr(1))
		enc.Send(selSetFragmentBytes, unsafe.Pointer(&push), uintptr(unsafe.Sizeof(push)), uintptr(1))
		enc.Send(selDraw, uintptr(primStrip), uintptr(0), uintptr(4), uintptr(fills))
	}
	if inked {
		s.blit(enc, s.inkBuf, width, height)
	}
	enc.Send(selEndEncoding)
	cmd.Send(selPresent, drawable)
	cmd.Send(selCommit)
	cmd.Send(selWait)
	return nil
}

func (s *Screen) blit(enc, buf objc.ID, width, height int) {
	push := inkPush{sizeX: int32(width), sizeY: int32(height), swapRB: channelSwap}
	enc.Send(selSetPipeline, s.ink)
	enc.Send(selSetFragmentBuffer, buf, uintptr(0), uintptr(0))
	enc.Send(selSetFragmentBytes, unsafe.Pointer(&push), uintptr(unsafe.Sizeof(push)), uintptr(1))
	enc.Send(selDraw, uintptr(primStrip), uintptr(0), uintptr(4), uintptr(1))
}

func (s *Screen) grow(slot *objc.ID, cap *int, raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if *slot != 0 && *cap >= len(raw) {
		ptr := uintptr((*slot).Send(selContents))
		if ptr == 0 {
			return ErrUnavailable
		}
		copy(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(raw)), raw)
		return nil
	}
	if *slot != 0 {
		(*slot).Send(selRelease)
		*slot = 0
	}
	buf := s.device.Send(selNewBuffer, unsafe.Pointer(&raw[0]), uintptr(len(raw)), uintptr(0))
	if buf == 0 {
		return ErrUnavailable
	}
	*slot = buf
	*cap = len(raw)
	return nil
}

// Close releases the device objects. The view stays with the caller.
func (s *Screen) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	uiDo(func() {
		withPool(func() {
			if s.layer != 0 {
				s.layer.Send(selRemove)
			}
			for _, id := range []objc.ID{s.fillBuf, s.underBuf, s.inkBuf, s.clear, s.fill, s.ink, s.lib, s.layer, s.queue, s.device} {
				if id != 0 {
					id.Send(selRelease)
				}
			}
		})
	})
	s.fillBuf, s.underBuf, s.inkBuf = 0, 0, 0
	s.clear, s.fill, s.ink, s.lib, s.layer, s.queue, s.device = 0, 0, 0, 0, 0, 0, 0
	s.view = 0
	return nil
}

type fillPush struct {
	extentX, extentY float32
	swapRB, pad      int32
}

type inkPush struct {
	sizeX, sizeY int32
	swapRB, pad  int32
}

type cgPoint struct{ X, Y float64 }
type cgSize struct{ Width, Height float64 }
type cgRect struct {
	Origin cgPoint
	Size   cgSize
}

func compileLibrary(device objc.ID) (objc.ID, error) {
	if newLib == nil {
		native.Register(&newLib, msgSend())
	}
	var err objc.ID
	lib := newLib(device, selNewLibrary, nsstr(ShaderSource), 0, &err)
	if lib == 0 {
		detail := ""
		if err != 0 {
			detail = native.GoString(uintptr(err.Send(selLocalized).Send(selUTF8)))
		}
		if detail == "" {
			return 0, fmt.Errorf("%w: library", ErrUnavailable)
		}
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, detail)
	}
	return lib, nil
}

func (s *Screen) pipeline(vert, frag string, blend bool) (objc.ID, error) {
	vs := s.lib.Send(selNewFunction, nsstr(vert))
	fs := s.lib.Send(selNewFunction, nsstr(frag))
	if vs == 0 || fs == 0 {
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, vert)
	}
	desc := objc.ID(objc.GetClass("MTLRenderPipelineDescriptor")).Send(selNew)
	if desc == 0 {
		return 0, ErrUnavailable
	}
	desc.Send(selSetVertexFn, vs)
	desc.Send(selSetFragmentFn, fs)
	att := desc.Send(selColorAttachments).Send(selSubscript, uintptr(0))
	att.Send(selSetPixelFormat, uintptr(pixelBGRA))
	if blend {
		att.Send(selSetBlend, true)
		att.Send(selSetSrcRGB, uintptr(blendOne))
		att.Send(selSetDstRGB, uintptr(blendOneMinus))
		att.Send(selSetSrcA, uintptr(blendOne))
		att.Send(selSetDstA, uintptr(blendOneMinus))
	}
	if newPipe == nil {
		native.Register(&newPipe, msgSend())
	}
	var err objc.ID
	pipe := newPipe(s.device, selNewPipe, desc, &err)
	desc.Send(selRelease)
	vs.Send(selRelease)
	fs.Send(selRelease)
	if pipe == 0 {
		detail := ""
		if err != 0 {
			detail = native.GoString(uintptr(err.Send(selLocalized).Send(selUTF8)))
		}
		if detail == "" {
			return 0, fmt.Errorf("%w: %s", ErrUnavailable, vert)
		}
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, detail)
	}
	return pipe, nil
}

// withPool runs fn inside one NSAutoreleasePool.
// The pool belongs to the OS thread that creates it. Lock around the
// push, the work, and the drain so a goroutine migration cannot drain
// it on another thread.
func withPool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	fn()
}

func nsstr(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(selUTF8String, s)
}

func boundsOf(view objc.ID) cgRect {
	if boundsFn == nil {
		native.Register(&boundsFn, msgSend())
	}
	return boundsFn(view, selBounds)
}

func setRect(id objc.ID, sel objc.SEL, rect cgRect) {
	if sendRect == nil {
		native.Register(&sendRect, msgSend())
	}
	sendRect(id, sel, rect)
}

func setSize(id objc.ID, sel objc.SEL, size cgSize) {
	if sendSize == nil {
		native.Register(&sendSize, msgSend())
	}
	sendSize(id, sel, size)
}

var (
	msgOnce  sync.Once
	msgAddr  uintptr
	newLib   func(objc.ID, objc.SEL, objc.ID, objc.ID, *objc.ID) objc.ID
	newPipe  func(objc.ID, objc.SEL, objc.ID, *objc.ID) objc.ID
	boundsFn func(objc.ID, objc.SEL) cgRect
	sendRect func(objc.ID, objc.SEL, cgRect)
	sendSize func(objc.ID, objc.SEL, cgSize)
)

func msgSend() uintptr {
	msgOnce.Do(func() {
		lib, err := native.Open("/usr/lib/libobjc.A.dylib", native.Lazy)
		if err != nil {
			return
		}
		addr, err := native.Symbol(lib, "objc_msgSend")
		if err != nil {
			return
		}
		msgAddr = addr
	})
	return msgAddr
}

var (
	selRetain             = objc.RegisterName("retain")
	selRelease            = objc.RegisterName("release")
	selNew                = objc.RegisterName("new")
	selAlloc              = objc.RegisterName("alloc")
	selInit               = objc.RegisterName("init")
	selDrain              = objc.RegisterName("drain")
	selLayer              = objc.RegisterName("layer")
	selNewCommandQueue    = objc.RegisterName("newCommandQueue")
	selNewLibrary         = objc.RegisterName("newLibraryWithSource:options:error:")
	selNewFunction        = objc.RegisterName("newFunctionWithName:")
	selNewPipe            = objc.RegisterName("newRenderPipelineStateWithDescriptor:error:")
	selNewBuffer          = objc.RegisterName("newBufferWithBytes:length:options:")
	selSetVertexFn        = objc.RegisterName("setVertexFunction:")
	selSetFragmentFn      = objc.RegisterName("setFragmentFunction:")
	selColorAttachments   = objc.RegisterName("colorAttachments")
	selSubscript          = objc.RegisterName("objectAtIndexedSubscript:")
	selSetPixelFormat     = objc.RegisterName("setPixelFormat:")
	selSetBlend           = objc.RegisterName("setBlendingEnabled:")
	selSetSrcRGB          = objc.RegisterName("setSourceRGBBlendFactor:")
	selSetDstRGB          = objc.RegisterName("setDestinationRGBBlendFactor:")
	selSetSrcA            = objc.RegisterName("setSourceAlphaBlendFactor:")
	selSetDstA            = objc.RegisterName("setDestinationAlphaBlendFactor:")
	selLocalized          = objc.RegisterName("localizedDescription")
	selUTF8               = objc.RegisterName("UTF8String")
	selUTF8String         = objc.RegisterName("stringWithUTF8String:")
	selSetDevice          = objc.RegisterName("setDevice:")
	selSetFramebufferOnly = objc.RegisterName("setFramebufferOnly:")
	selSetOpaque          = objc.RegisterName("setOpaque:")
	selSetDisplaySync     = objc.RegisterName("setDisplaySyncEnabled:")
	selSetWantsLayer      = objc.RegisterName("setWantsLayer:")
	selSetLayer           = objc.RegisterName("setLayer:")
	selAddSublayer        = objc.RegisterName("addSublayer:")
	selRemove             = objc.RegisterName("removeFromSuperlayer")
	selSetFrame           = objc.RegisterName("setFrame:")
	selBounds             = objc.RegisterName("bounds")
	selSetDrawableSize    = objc.RegisterName("setDrawableSize:")
	selNextDrawable       = objc.RegisterName("nextDrawable")
	selCommandBuffer      = objc.RegisterName("commandBuffer")
	selRenderPass         = objc.RegisterName("renderPassDescriptor")
	selSetTexture         = objc.RegisterName("setTexture:")
	selSetLoadAction      = objc.RegisterName("setLoadAction:")
	selSetStoreAction     = objc.RegisterName("setStoreAction:")
	selTexture            = objc.RegisterName("texture")
	selEncoder            = objc.RegisterName("renderCommandEncoderWithDescriptor:")
	selSetPipeline        = objc.RegisterName("setRenderPipelineState:")
	selSetVertexBuffer    = objc.RegisterName("setVertexBuffer:offset:atIndex:")
	selSetFragmentBuffer  = objc.RegisterName("setFragmentBuffer:offset:atIndex:")
	selSetVertexBytes     = objc.RegisterName("setVertexBytes:length:atIndex:")
	selSetFragmentBytes   = objc.RegisterName("setFragmentBytes:length:atIndex:")
	selDraw               = objc.RegisterName("drawPrimitives:vertexStart:vertexCount:instanceCount:")
	selEndEncoding        = objc.RegisterName("endEncoding")
	selPresent            = objc.RegisterName("presentDrawable:")
	selCommit             = objc.RegisterName("commit")
	selWait               = objc.RegisterName("waitUntilCompleted")
	selContents           = objc.RegisterName("contents")
)
