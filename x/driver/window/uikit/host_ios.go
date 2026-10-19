//go:build ios

package uikit

import (
	"fmt"
	"log/slog"
	"sync"
	"unicode/utf8"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	surfaceOnce sync.Once
	surfaceErr  error
	surfaceKind objc.Class
	keptWindow  objc.ID

	pointFn func(objc.ID, objc.SEL, objc.ID) cgPoint

	selInit         = objc.RegisterName("init")
	selSetRoot      = objc.RegisterName("setRootViewController:")
	selMakeVisible  = objc.RegisterName("makeKeyAndVisible")
	selInitScene    = objc.RegisterName("initWithWindowScene:")
	selIsKind       = objc.RegisterName("isKindOfClass:")
	selAny          = objc.RegisterName("anyObject")
	selLocation     = objc.RegisterName("locationInView:")
	selLayout       = objc.RegisterName("layoutSubviews")
	selSafeChanged  = objc.RegisterName("safeAreaInsetsDidChange")
	selCanFocus     = objc.RegisterName("canBecomeFirstResponder")
	selTouchesBegan = objc.RegisterName("touchesBegan:withEvent:")
	selTouchesMoved = objc.RegisterName("touchesMoved:withEvent:")
	selTouchesEnded = objc.RegisterName("touchesEnded:withEvent:")
	selTouchesStop  = objc.RegisterName("touchesCancelled:withEvent:")
	selPressBegan   = objc.RegisterName("pressesBegan:withEvent:")
	selPressEnded   = objc.RegisterName("pressesEnded:withEvent:")
	selPressStop    = objc.RegisterName("pressesCancelled:withEvent:")
	selKey          = objc.RegisterName("key")
	selCharacters   = objc.RegisterName("characters")
	selKeyCode      = objc.RegisterName("keyCode")
)

func openHost(app objc.ID) (objc.ID, error) {
	if app == 0 {
		return 0, fmt.Errorf("%w: UIApplication", window.ErrInit)
	}
	bounds := boundsOf(objc.ID(objc.GetClass("UIScreen")).Send(selMainScreen))
	if bounds.Size.Width < 1 || bounds.Size.Height < 1 {
		bounds.Size.Width = 320
		bounds.Size.Height = 480
	}
	host := objc.ID(0)
	if scene := windowScene(app); scene != 0 {
		host = objc.ID(objc.GetClass("UIWindow")).Send(selAlloc).Send(selInitScene, scene)
	}
	if host == 0 {
		host = objc.ID(objc.GetClass("UIWindow")).Send(selAlloc).Send(selInitWithFrame, bounds)
	}
	if host == 0 {
		return 0, fmt.Errorf("%w: UIWindow", window.ErrInit)
	}
	if color := objc.ID(objc.GetClass("UIColor")).Send(selBlackColor); color != 0 {
		host.Send(selSetBackground, color)
	}
	root := objc.ID(objc.GetClass("UIViewController")).Send(selAlloc).Send(selInit)
	if root == 0 {
		host.Send(selRelease)
		return 0, fmt.Errorf("%w: controller", window.ErrInit)
	}
	host.Send(selSetRoot, root)
	root.Send(selRelease)
	host.Send(selMakeVisible)
	keptWindow = host
	slog.Info("ios window")
	return host, nil
}

func windowScene(app objc.ID) objc.ID {
	items := app.Send(selConnectedScenes).Send(selAllObjects)
	if items == 0 {
		return 0
	}
	kind := objc.ID(objc.GetClass("UIWindowScene"))
	if kind == 0 {
		return 0
	}
	n := int(items.Send(selCount))
	if n > 8 {
		n = 8
	}
	for i := range n {
		scene := items.Send(selObjectAtIndex, uintptr(i))
		if scene != 0 && scene.Send(selIsKind, kind) != 0 {
			return scene
		}
	}
	return 0
}

func surfaceClass() objc.Class {
	surfaceOnce.Do(func() {
		surfaceKind, surfaceErr = objc.RegisterClass(
			"LewkitSurface",
			objc.GetClass("UIView"),
			nil,
			nil,
			[]objc.MethodDef{
				{Cmd: selLayout, Fn: surfaceLayout},
				{Cmd: selSafeChanged, Fn: surfaceSafe},
				{Cmd: selCanFocus, Fn: surfaceCanFocus},
				{Cmd: selTouchesBegan, Fn: surfaceTouchDown},
				{Cmd: selTouchesMoved, Fn: surfaceTouchMove},
				{Cmd: selTouchesEnded, Fn: surfaceTouchUp},
				{Cmd: selTouchesStop, Fn: surfaceTouchUp},
				{Cmd: selPressBegan, Fn: surfaceKeyDown},
				{Cmd: selPressEnded, Fn: surfaceKeyUp},
				{Cmd: selPressStop, Fn: surfaceKeyUp},
			},
		)
	})
	if surfaceErr != nil {
		return 0
	}
	return surfaceKind
}

func surfaceCanFocus(objc.ID, objc.SEL) bool { return true }

func surfaceLayout(self objc.ID, cmd objc.SEL) {
	self.SendSuper(cmd)
	reportSurface(self)
}

func surfaceSafe(self objc.ID, cmd objc.SEL) {
	self.SendSuper(cmd)
	reportSurface(self)
}

func surfaceTouchDown(self objc.ID, _ objc.SEL, touches, event objc.ID) {
	surfaceTouch(self, touches, event, 0)
}

func surfaceTouchMove(self objc.ID, _ objc.SEL, touches, event objc.ID) {
	surfaceTouch(self, touches, event, 2)
}

func surfaceTouchUp(self objc.ID, _ objc.SEL, touches, event objc.ID) {
	surfaceTouch(self, touches, event, 1)
}

func surfaceTouch(self, touches, _ objc.ID, action int) {
	if self == 0 || touches == 0 {
		return
	}
	touch := touches.Send(selAny)
	if touch == 0 {
		return
	}
	if pointFn == nil {
		native.Register(&pointFn, native.MsgSend())
	}
	pt := pointFn(touch, selLocation, self)
	scale := screenScale(self.Send(selWindow))
	entry.DeliverPointer(int(pt.X*scale+0.5), int(pt.Y*scale+0.5), action)
}

func surfaceKeyDown(self objc.ID, _ objc.SEL, presses, event objc.ID) {
	surfaceKey(self, presses, event, true)
}

func surfaceKeyUp(self objc.ID, _ objc.SEL, presses, event objc.ID) {
	surfaceKey(self, presses, event, false)
}

func surfaceKey(_, presses, _ objc.ID, pressed bool) {
	if presses == 0 {
		return
	}
	press := presses.Send(selAny)
	if press == 0 {
		return
	}
	key := press.Send(selKey)
	if key == 0 {
		return
	}
	r, code := hidKey(int(key.Send(selKeyCode)))
	if text := native.UTF8String(key.Send(selCharacters)); text != "" {
		r, _ = utf8.DecodeRuneInString(text)
	}
	if r == 0 && code == 0 {
		return
	}
	entry.DeliverKey(r, code, pressed)
}

func reportSurface(self objc.ID) {
	if self == 0 {
		return
	}
	scale := screenScale(self.Send(selWindow))
	rect := boundsOf(self)
	width := int(rect.Size.Width*scale + 0.5)
	height := int(rect.Size.Height*scale + 0.5)
	if width < 1 || height < 1 {
		return
	}
	entry.DeliverResize(width, height)
	insets := edgeInsetsOf(self)
	entry.DeliverInsets(
		int(insets.Left*scale+0.5),
		int(insets.Top*scale+0.5),
		int(insets.Right*scale+0.5),
		int(insets.Bottom*scale+0.5),
		width, height,
	)
}

// hidKey maps a UIKeyboardHIDUsage to the key codes the GUI examples already read.
// Arrows and delete often arrive with an empty characters string.
func hidKey(hid int) (rune, uint32) {
	switch hid {
	case 0x2A:
		return 127, 51
	case 0x28:
		return '\r', 36
	case 0x50:
		return 0, 123
	case 0x4F:
		return 0, 124
	case 0x52:
		return 0, 126
	case 0x51:
		return 0, 125
	default:
		return 0, 0
	}
}
