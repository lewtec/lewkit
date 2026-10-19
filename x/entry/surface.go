package entry

import (
	"sync"
	"sync/atomic"
)

var (
	pointerFn     atomic.Value
	resizeFn      atomic.Value
	surfaceLostFn atomic.Value
	keyFn         atomic.Value
)

type insetSample struct {
	left, top, right, bottom int
	width, height            int
	set                      bool
}

var (
	insetMu  sync.Mutex
	insetFn  func(left, top, right, bottom, width, height int)
	insetNow insetSample
)

// HandlePointer receives touch samples. action is 0 down, 1 up, 2 move.
func HandlePointer(fn func(x, y, action int)) {
	if fn == nil {
		return
	}
	pointerFn.Store(fn)
}

// HandleKey receives one hardware key. code is the platform key code.
// pressed is false on release.
func HandleKey(fn func(r rune, code uint32, pressed bool)) {
	if fn == nil {
		return
	}
	keyFn.Store(fn)
}

// DeliverKey runs the handler registered with [HandleKey].
func DeliverKey(r rune, code uint32, pressed bool) {
	fn, _ := keyFn.Load().(func(rune, uint32, bool))
	if fn != nil {
		fn(r, code, pressed)
	}
}

// DeliverPointer runs the handler registered with [HandlePointer].
func DeliverPointer(x, y, action int) {
	fn, _ := pointerFn.Load().(func(int, int, int))
	if fn != nil {
		fn(x, y, action)
	}
}

// HandleResize receives a new surface size in pixels.
func HandleResize(fn func(width, height int)) {
	if fn == nil {
		return
	}
	resizeFn.Store(fn)
}

// DeliverResize runs the handler registered with [HandleResize].
func DeliverResize(width, height int) {
	fn, _ := resizeFn.Load().(func(int, int))
	if fn != nil {
		fn(width, height)
	}
}

// HandleSurfaceLost runs when the host destroys the current surface.
func HandleSurfaceLost(fn func()) {
	if fn == nil {
		return
	}
	surfaceLostFn.Store(fn)
}

// DeliverSurfaceLost runs the handler registered with [HandleSurfaceLost].
func DeliverSurfaceLost() {
	fn, _ := surfaceLostFn.Load().(func())
	if fn != nil {
		fn()
	}
}

var fontScaleFn atomic.Value

// HandleFontScale receives the user's text-size preference.
// One is the default. The value does not include display density.
func HandleFontScale(fn func(float64)) {
	if fn == nil {
		return
	}
	fontScaleFn.Store(fn)
}

// DeliverFontScale runs the handler registered with [HandleFontScale].
func DeliverFontScale(scale float64) {
	fn, _ := fontScaleFn.Load().(func(float64))
	if fn != nil {
		fn(scale)
	}
}

// HandleInsets receives the host safe area. The arguments are pixels:
// left, top, right, and bottom insets, then the client width and height.
// A sample that arrived before the handler is delivered when it is registered.
func HandleInsets(fn func(left, top, right, bottom, width, height int)) {
	if fn == nil {
		return
	}
	insetMu.Lock()
	insetFn = fn
	sample := insetNow
	insetMu.Unlock()
	if sample.set {
		fn(sample.left, sample.top, sample.right, sample.bottom, sample.width, sample.height)
	}
}

// DeliverInsets records the latest host safe area and runs the handler.
func DeliverInsets(left, top, right, bottom, width, height int) {
	insetMu.Lock()
	insetNow = insetSample{left, top, right, bottom, width, height, true}
	fn := insetFn
	insetMu.Unlock()
	if fn != nil {
		fn(left, top, right, bottom, width, height)
	}
}
