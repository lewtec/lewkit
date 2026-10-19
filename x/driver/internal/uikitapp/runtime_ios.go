//go:build ios

package uikitapp

import (
	"context"
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	uiKitPath      = "/System/Library/Frameworks/UIKit.framework/UIKit"
	foundationPath = "/System/Library/Frameworks/Foundation.framework/Foundation"
	notifyPath     = "/System/Library/Frameworks/UserNotifications.framework/UserNotifications"
	typePath       = "/System/Library/Frameworks/UniformTypeIdentifiers.framework/UniformTypeIdentifiers"
)

type cgPoint struct{ X, Y float64 }
type cgSize struct{ Width, Height float64 }
type cgRect struct {
	Origin cgPoint
	Size   cgSize
}

var (
	uiOnce   sync.Once
	uiErr    error
	bindOnce sync.Once

	boundsFn func(objc.ID, objc.SEL) cgRect
	rectFn   func(objc.ID, objc.SEL, cgRect)
	menuFn   func(objc.ID, objc.SEL, cgRect, objc.ID, bool) bool
	levelFn  func(objc.ID, objc.SEL) float64

	gateMu sync.Mutex
	gateOn bool

	shownMu        sync.Mutex
	shownPresenter objc.ID

	selAlloc     = objc.RegisterName("alloc")
	selInit      = objc.RegisterName("init")
	selNew       = objc.RegisterName("new")
	selRetain    = objc.RegisterName("retain")
	selRelease   = objc.RegisterName("release")
	selDrain     = objc.RegisterName("drain")
	selShared    = objc.RegisterName("sharedApplication")
	selConnected = objc.RegisterName("connectedScenes")
	selAll       = objc.RegisterName("allObjects")
	selCount     = objc.RegisterName("count")
	selAt        = objc.RegisterName("objectAtIndex:")
	selWindows   = objc.RegisterName("windows")
	selIsKey     = objc.RegisterName("isKeyWindow")
	selKeyWindow = objc.RegisterName("keyWindow")
	selRoot      = objc.RegisterName("rootViewController")
	selPresented = objc.RegisterName("presentedViewController")
	selView      = objc.RegisterName("view")
	selBounds    = objc.RegisterName("bounds")
	selPresent   = objc.RegisterName("presentViewController:animated:completion:")
	selDismiss   = objc.RegisterName("dismissViewControllerAnimated:completion:")
	selPopover   = objc.RegisterName("popoverPresentationController")
	selSetSource = objc.RegisterName("setSourceView:")
	selSetRect   = objc.RegisterName("setSourceRect:")
	selSetArrows = objc.RegisterName("setPermittedArrowDirections:")
	selResponds  = objc.RegisterName("respondsToSelector:")
	selInstances = objc.RegisterName("instancesRespondToSelector:")
	selDesc      = objc.RegisterName("localizedDescription")
	selArray     = objc.RegisterName("array")
	selArrayOne  = objc.RegisterName("arrayWithObject:")
	selAdd       = objc.RegisterName("addObject:")
	selFirst     = objc.RegisterName("firstObject")
	selFileURL   = objc.RegisterName("fileURLWithPath:")
	selURLString = objc.RegisterName("URLWithString:")
	selPath      = objc.RegisterName("path")
	selScheme    = objc.RegisterName("scheme")
	selIsFile    = objc.RegisterName("isFileURL")
)

func cls(name string) objc.ID {
	return objc.ID(objc.GetClass(name))
}

func ensure() error {
	uiOnce.Do(func() {
		if _, err := native.Open(foundationPath, native.Global|native.Lazy); err != nil {
			uiErr = fmt.Errorf("%w: Foundation", driver.ErrUnavailable)
			return
		}
		if _, err := native.Open(uiKitPath, native.Global|native.Lazy); err != nil {
			uiErr = fmt.Errorf("%w: UIKit", driver.ErrUnavailable)
			return
		}
		_, _ = native.Open(notifyPath, native.Global|native.Lazy)
		_, _ = native.Open(typePath, native.Global|native.Lazy)
		if native.MsgSend() == 0 {
			uiErr = fmt.Errorf("%w: objc", driver.ErrUnavailable)
			return
		}
		bindOnce.Do(func() {
			native.Register(&boundsFn, native.MsgSend())
			native.Register(&rectFn, native.MsgSend())
			native.Register(&menuFn, native.MsgSend())
			native.Register(&levelFn, native.MsgSend())
		})
	})
	return uiErr
}

func withPool(fn func()) {
	pool := cls("NSAutoreleasePool").Send(selAlloc).Send(selInit)
	defer pool.Send(selDrain)
	fn()
}

func onUI(fn func()) error {
	if !thread.Bound() {
		return fmt.Errorf("%w: ui thread", driver.ErrUnavailable)
	}
	if thread.On() {
		fn()
		return nil
	}
	thread.Do(fn)
	return nil
}

func lockGate() bool {
	gateMu.Lock()
	defer gateMu.Unlock()
	if gateOn {
		return false
	}
	gateOn = true
	return true
}

func unlockGate() {
	gateMu.Lock()
	gateOn = false
	gateMu.Unlock()
}

func remember(presenter objc.ID) {
	shownMu.Lock()
	shownPresenter = presenter
	shownMu.Unlock()
}

func clearShown() {
	shownMu.Lock()
	shownPresenter = 0
	shownMu.Unlock()
}

func dismissShown() {
	shownMu.Lock()
	presenter := shownPresenter
	shownPresenter = 0
	shownMu.Unlock()
	if presenter != 0 {
		presenter.Send(selDismiss, false, objc.ID(0))
	}
}

func hook(done func(string)) func(string) {
	var once sync.Once
	return func(reply string) {
		once.Do(func() {
			clearShown()
			done(reply)
		})
	}
}

// wait runs present on the UI thread and blocks until done or ctx ends.
// present must return without waiting for a tap. A nested run loop never delivers it.
func wait(ctx context.Context, present func(done func(string))) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if thread.Bound() && thread.On() {
		return "", fmt.Errorf("%w: ui thread", driver.ErrUnavailable)
	}
	answer := make(chan string, 1)
	var setup error
	err := onUI(func() {
		if err := ensure(); err != nil {
			setup = err
			return
		}
		present(func(reply string) {
			select {
			case answer <- reply:
			default:
			}
		})
	})
	if err != nil {
		return "", err
	}
	if setup != nil {
		return "", setup
	}
	select {
	case <-ctx.Done():
		_ = onUI(dismissShown)
		return "", context.Cause(ctx)
	case reply := <-answer:
		return reply, nil
	}
}

func topController() objc.ID {
	app := cls("UIApplication").Send(selShared)
	win := keyWindow(app)
	if win == 0 {
		return 0
	}
	root := win.Send(selRoot)
	for root != 0 {
		next := root.Send(selPresented)
		if next == 0 {
			return root
		}
		root = next
	}
	return 0
}

func keyWindow(app objc.ID) objc.ID {
	if app == 0 {
		return 0
	}
	scenes := app.Send(selConnected).Send(selAll)
	n := int(scenes.Send(selCount))
	if n > 8 {
		n = 8
	}
	var fallback objc.ID
	for i := range n {
		scene := scenes.Send(selAt, uintptr(i))
		wins := scene.Send(selWindows)
		wn := int(wins.Send(selCount))
		if wn > 8 {
			wn = 8
		}
		for j := range wn {
			win := wins.Send(selAt, uintptr(j))
			if win == 0 {
				continue
			}
			if fallback == 0 {
				fallback = win
			}
			if win.Send(selIsKey) != 0 {
				return win
			}
		}
	}
	if key := app.Send(selKeyWindow); key != 0 {
		return key
	}
	return fallback
}

func viewBounds(id objc.ID) cgRect {
	if id == 0 || boundsFn == nil {
		return cgRect{}
	}
	return boundsFn(id, selBounds)
}

func pinPopover(ctrl, presenter objc.ID) {
	pop := ctrl.Send(selPopover)
	if pop == 0 || presenter == 0 || rectFn == nil {
		return
	}
	view := presenter.Send(selView)
	if view == 0 {
		return
	}
	pop.Send(selSetSource, view)
	rect := viewBounds(view)
	rectFn(pop, selSetRect, cgRect{
		Origin: cgPoint{X: rect.Size.Width / 2, Y: rect.Size.Height / 2},
		Size:   cgSize{Width: 1, Height: 1},
	})
	pop.Send(selSetArrows, uintptr(0))
}

func present(presenter, ctrl objc.ID) {
	if presenter == 0 || ctrl == 0 {
		return
	}
	remember(presenter)
	pinPopover(ctrl, presenter)
	presenter.Send(selPresent, ctrl, true, objc.ID(0))
}

func errorText(id objc.ID) string {
	if id == 0 {
		return ""
	}
	return native.UTF8String(id.Send(selDesc))
}

func ns(text string) objc.ID { return native.NSString(text) }

func objcText(id objc.ID) string { return native.UTF8String(id) }

func responds(id objc.ID, sel objc.SEL) bool {
	return id != 0 && id.Send(selResponds, sel) != 0
}

func instances(class objc.ID, sel objc.SEL) bool {
	return class != 0 && class.Send(selInstances, sel) != 0
}
