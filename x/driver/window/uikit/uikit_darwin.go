//go:build darwin

package uikit

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"sync"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/ffi/native/dispatch"
)

// surfaceTag marks the view the iOS host checks before covering it with the splash.
const surfaceTag = 0x6C6577

func open(ctx context.Context, cfg window.Config) (window.Window, error) {
	if runtime.GOOS != "ios" {
		return nil, fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	width, height, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	buf := window.NewBuffer(width, height)
	buf.SetFramePeriod(cfg.Period)
	w := &win{Buffer: buf}
	entry.HandlePointer(func(x, y, action int) {
		w.Emit(window.TouchPointer(x, y, action))
	})
	entry.HandleResize(func(pw, ph int) {
		if pw < 1 || ph < 1 {
			return
		}
		_ = w.Resize(image.Pt(pw, ph))
	})
	entry.HandleSurfaceLost(func() {
		w.mu.Lock()
		w.view = 0
		w.mu.Unlock()
	})
	var view uintptr
	var pw, ph int
	var period time.Duration
	dispatch.OnMain(func() {
		view, pw, ph, period, err = attach()
	})
	if err != nil {
		return nil, err
	}
	if cfg.Period <= 0 && period > 0 {
		w.SetFramePeriod(period)
	}
	if pw > 0 && ph > 0 {
		_, _ = w.EnsureSize(image.Pt(pw, ph))
	}
	w.mu.Lock()
	w.view = view
	w.mu.Unlock()
	return w, nil
}

type win struct {
	*window.Buffer
	mu   sync.Mutex
	view uintptr
}

func (w *win) Draw() error { return w.Swap() }

func (w *win) Surface() window.Surface {
	if w == nil {
		return window.Surface{}
	}
	w.mu.Lock()
	view := w.view
	w.mu.Unlock()
	return window.Surface{Kind: window.SurfaceUIView, A: view}
}

func (w *win) Close() error {
	w.mu.Lock()
	view := w.view
	w.view = 0
	w.mu.Unlock()
	if view != 0 {
		dispatch.OnMain(func() {
			objc.ID(view).Send(selRemoveFromSuperview)
		})
	}
	return w.Buffer.Close()
}

func attach() (uintptr, int, int, time.Duration, error) {
	if _, err := native.Open("/System/Library/Frameworks/UIKit.framework/UIKit", native.Global|native.Lazy); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("%w: UIKit", driver.ErrIncompatible)
	}
	var view objc.ID
	var width, height int
	var period time.Duration
	var err error
	withPool(func() {
		app := objc.ID(objc.GetClass("UIApplication")).Send(selSharedApplication)
		host := keyWindow(app)
		if host == 0 {
			err = fmt.Errorf("%w: no window", window.ErrInit)
			return
		}
		period = framePeriod(host)
		root := host.Send(selRootViewController)
		if root != 0 && responds(root, selOpenSurface) {
			view = root.Send(selOpenSurface)
		}
		if view == 0 {
			view, err = cover(host)
			if err != nil {
				return
			}
		}
		scale := screenScale(host)
		rect := boundsOf(view)
		width = int(rect.Size.Width*scale + 0.5)
		height = int(rect.Size.Height*scale + 0.5)
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if view == 0 {
		return 0, 0, 0, 0, fmt.Errorf("%w: no view", window.ErrInit)
	}
	return uintptr(view), width, height, period, nil
}

func keyWindow(app objc.ID) objc.ID {
	if app == 0 {
		return 0
	}
	scenes := app.Send(selConnectedScenes).Send(selAllObjects)
	n := int(scenes.Send(selCount))
	var fallback objc.ID
	for i := 0; i < n; i++ {
		scene := scenes.Send(selObjectAtIndex, uintptr(i))
		wins := scene.Send(selWindows)
		if wins == 0 {
			continue
		}
		wn := int(wins.Send(selCount))
		for j := 0; j < wn; j++ {
			win := wins.Send(selObjectAtIndex, uintptr(j))
			if fallback == 0 {
				fallback = win
			}
			if win.Send(selIsKeyWindow) != 0 {
				return win
			}
		}
	}
	if key := app.Send(selKeyWindow); key != 0 {
		return key
	}
	return fallback
}

func cover(host objc.ID) (objc.ID, error) {
	root := host.Send(selRootViewController)
	parent := objc.ID(0)
	if root != 0 {
		parent = root.Send(selView)
	}
	if parent == 0 {
		return 0, fmt.Errorf("%w: no root view", window.ErrInit)
	}
	view := objc.ID(objc.GetClass("UIView")).Send(selAlloc).Send(selInitWithFrame, boundsOf(parent))
	if view == 0 {
		return 0, fmt.Errorf("%w: UIView", window.ErrInit)
	}
	view.Send(selSetTag, uintptr(surfaceTag))
	view.Send(selSetAutoresizing, uintptr(18))
	parent.Send(selAddSubview, view)
	parent.Send(selBringSubviewToFront, view)
	return view, nil
}

func screenScale(host objc.ID) float64 {
	screen := host.Send(selScreen)
	if screen == 0 {
		screen = objc.ID(objc.GetClass("UIScreen")).Send(selMainScreen)
	}
	if screen == 0 {
		return 1
	}
	if scaleFn == nil {
		native.Register(&scaleFn, msgSend())
	}
	scale := scaleFn(screen, selScale)
	if scale < 1 {
		return 1
	}
	return scale
}

func framePeriod(host objc.ID) time.Duration {
	screen := host.Send(selScreen)
	if screen == 0 {
		return 0
	}
	fps := int(screen.Send(selMaximumFrames))
	if fps <= 0 {
		return 0
	}
	return time.Second / time.Duration(fps)
}

func responds(id objc.ID, sel objc.SEL) bool {
	if respondsFn == nil {
		native.Register(&respondsFn, msgSend())
	}
	return respondsFn(id, selResponds, sel)
}

func withPool(fn func()) {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	fn()
}

func boundsOf(view objc.ID) cgRect {
	if boundsFn == nil {
		native.Register(&boundsFn, msgSend())
	}
	return boundsFn(view, selBounds)
}

type cgPoint struct{ X, Y float64 }
type cgSize struct{ Width, Height float64 }
type cgRect struct {
	Origin cgPoint
	Size   cgSize
}

var (
	msgOnce    sync.Once
	msgAddr    uintptr
	boundsFn   func(objc.ID, objc.SEL) cgRect
	scaleFn    func(objc.ID, objc.SEL) float64
	respondsFn func(objc.ID, objc.SEL, objc.SEL) bool
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
	selSharedApplication   = objc.RegisterName("sharedApplication")
	selConnectedScenes     = objc.RegisterName("connectedScenes")
	selAllObjects          = objc.RegisterName("allObjects")
	selCount               = objc.RegisterName("count")
	selObjectAtIndex       = objc.RegisterName("objectAtIndex:")
	selWindows             = objc.RegisterName("windows")
	selIsKeyWindow         = objc.RegisterName("isKeyWindow")
	selKeyWindow           = objc.RegisterName("keyWindow")
	selRootViewController  = objc.RegisterName("rootViewController")
	selOpenSurface         = objc.RegisterName("openSurface")
	selResponds            = objc.RegisterName("respondsToSelector:")
	selView                = objc.RegisterName("view")
	selAlloc               = objc.RegisterName("alloc")
	selInitWithFrame       = objc.RegisterName("initWithFrame:")
	selSetTag              = objc.RegisterName("setTag:")
	selSetAutoresizing     = objc.RegisterName("setAutoresizingMask:")
	selAddSubview          = objc.RegisterName("addSubview:")
	selBringSubviewToFront = objc.RegisterName("bringSubviewToFront:")
	selRemoveFromSuperview = objc.RegisterName("removeFromSuperview")
	selScreen              = objc.RegisterName("screen")
	selMainScreen          = objc.RegisterName("mainScreen")
	selScale               = objc.RegisterName("scale")
	selMaximumFrames       = objc.RegisterName("maximumFramesPerSecond")
	selBounds              = objc.RegisterName("bounds")
	selNew                 = objc.RegisterName("new")
	selDrain               = objc.RegisterName("drain")
)
