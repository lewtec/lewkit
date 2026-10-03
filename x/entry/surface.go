package entry

import "sync/atomic"

var (
	pointerFn     atomic.Value
	resizeFn      atomic.Value
	surfaceLostFn atomic.Value
)

// HandlePointer receives touch samples. action is 0 down, 1 up, 2 move.
func HandlePointer(fn func(x, y, action int)) {
	if fn == nil {
		return
	}
	pointerFn.Store(fn)
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
