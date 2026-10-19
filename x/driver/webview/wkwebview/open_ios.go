//go:build ios

package wkwebview

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	// UIViewAutoresizingFlexibleWidth | UIViewAutoresizingFlexibleHeight.
	viewStretches = 2 | 16
	// UIUserInterfaceStyleLight and UIUserInterfaceStyleDark.
	styleLight = 1
	styleDark  = 2
)

var (
	uiOnce sync.Once
	uiApp  objc.ID

	boundsMsg func(objc.ID, objc.SEL) nsRect

	selInitFrame      = objc.RegisterName("initWithFrame:")
	selSetRoot        = objc.RegisterName("setRootViewController:")
	selControllerView = objc.RegisterName("view")
	selSetBackground  = objc.RegisterName("setBackgroundColor:")
	selWhiteColor     = objc.RegisterName("whiteColor")
	selSetAutoresize  = objc.RegisterName("setAutoresizingMask:")
	selAddSubview     = objc.RegisterName("addSubview:")
	selMakeVisible    = objc.RegisterName("makeKeyAndVisible")
	selOverrideStyle  = objc.RegisterName("setOverrideUserInterfaceStyle:")
	selRemoveSubview  = objc.RegisterName("removeFromSuperview")
	selSetHidden      = objc.RegisterName("setHidden:")
	selMainScreen     = objc.RegisterName("mainScreen")
	selBounds         = objc.RegisterName("bounds")
	selPoolNew        = objc.RegisterName("new")
	selPoolDrain      = objc.RegisterName("drain")
	selCurrentRunLoop = objc.RegisterName("currentRunLoop")
	selRunUntil       = objc.RegisterName("runMode:beforeDate:")
	selFlush          = objc.RegisterName("flush")

	displayOnce sync.Once
)

func uiApplication() objc.ID {
	uiOnce.Do(func() {
		uiApp = currentUIApplication()
		if uiApp != 0 {
			return
		}
		// -[UIApplication init] raises when a singleton already exists.
		// UIApplicationMain's path publishes that singleton instead.
		instantiateUIApplication()
		uiApp = currentUIApplication()
	})
	return uiApp
}

func currentUIApplication() objc.ID {
	class := objc.GetClass("UIApplication")
	if class == 0 {
		return 0
	}
	if shared := objc.ID(class).Send(selShared); shared != 0 {
		return shared
	}
	return publishedUIApp()
}

func instantiateUIApplication() {
	handle, err := native.Open("/System/Library/Frameworks/UIKit.framework/UIKit", native.Global|native.Lazy)
	if err != nil {
		return
	}
	if sym, symErr := native.Symbol(handle, "UIApplicationInitialize"); symErr == nil && sym != 0 {
		var fn func()
		native.Register(&fn, sym)
		fn()
	}
	sym, err := native.Symbol(handle, "UIApplicationInstantiateSingleton")
	if err != nil || sym == 0 {
		return
	}
	var fn func(objc.Class)
	native.Register(&fn, sym)
	fn(objc.GetClass("UIApplication"))
}

func publishedUIApp() objc.ID {
	handle, err := native.Open("/System/Library/Frameworks/UIKit.framework/UIKit", native.Global|native.Lazy)
	if err != nil {
		return 0
	}
	sym, err := native.Symbol(handle, "UIApp")
	if err != nil || sym == 0 {
		return 0
	}
	return *(*objc.ID)(unsafe.Pointer(sym))
}

func (view *webKitView) createIOS(ctx context.Context, cfg webview.Config) error {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selPoolNew)
	defer pool.Send(selPoolDrain)
	width, height, err := cfg.Size()
	if err != nil {
		return err
	}
	if uiApplication() == 0 {
		return fmt.Errorf("%w: UIApplication", driver.ErrUnavailable)
	}
	bounds := mainScreenBounds()
	if bounds.Size.Width < 1 || bounds.Size.Height < 1 {
		bounds.Size.Width = float64(width)
		bounds.Size.Height = float64(height)
	}
	window := objc.ID(objc.GetClass("UIWindow")).Send(selAlloc).Send(selInitFrame, bounds)
	if window == 0 {
		return fmt.Errorf("%w: window", driver.ErrUnavailable)
	}
	if color := objc.ID(objc.GetClass("UIColor")).Send(selWhiteColor); color != 0 {
		window.Send(selSetBackground, color)
	}
	root := objc.ID(objc.GetClass("UIViewController")).Send(selAlloc).Send(selInit)
	if root == 0 {
		return fmt.Errorf("%w: controller", driver.ErrUnavailable)
	}
	window.Send(selSetRoot, root)
	host := root.Send(selControllerView)
	if host == 0 {
		return fmt.Errorf("%w: view", driver.ErrUnavailable)
	}
	handler := objc.ID(handlerClass).Send(selAlloc).Send(selInit)
	if handler == 0 {
		return fmt.Errorf("%w: handler", driver.ErrUnavailable)
	}
	view.nativeHandler = handler
	handlers.Store(uintptr(handler), view)
	configuration := objc.ID(objc.GetClass("WKWebViewConfiguration")).Send(selAlloc).Send(selInit)
	controller := configuration.Send(selUserContent)
	controller.Send(selAddHandler, handler, nsString(webview.ScriptName()))
	script := objc.ID(objc.GetClass("WKUserScript")).Send(selAlloc).Send(selInitScript, nsString(webview.WebKitBridge()), 0, true)
	controller.Send(selAddScript, script)
	configuration.Send(selSetScheme, handler, nsString(schemeName))
	// The default store is enough when this WebKit has no custom directory API.
	_ = applyProfile(configuration, cfg.Profile)
	frame := objectBounds(host)
	if frame.Size.Width < 1 || frame.Size.Height < 1 {
		frame = bounds
	}
	webView := objc.ID(objc.GetClass("WKWebView")).Send(selAlloc).Send(selInitWebView, frame, configuration)
	if webView == 0 {
		return fmt.Errorf("%w: WKWebView", driver.ErrUnavailable)
	}
	webView.Send(selSetAutoresize, viewStretches)
	host.Send(selAddSubview, webView)
	view.window = window
	view.webView = webView
	if scheme, err := daynight.Current(ctx); err == nil {
		view.useScheme(scheme, false)
	}
	window.Send(selMakeVisible)
	slog.Info("ios window")
	origin := fmt.Sprintf("%s://%s%d/", schemeName, viewHostPrefix, view.identifier)
	base := objc.ID(objc.GetClass("NSURL")).Send(selURLWithString, nsString(origin))
	if strings.TrimSpace(cfg.HTML) != "" {
		webView.Send(selLoadHTML, nsString(cfg.HTML), base)
		return nil
	}
	if cfg.Handler != nil {
		request := objc.ID(objc.GetClass("NSURLRequest")).Send(selRequestWithURL, base)
		webView.Send(selLoadRequest, request)
		return nil
	}
	page, err := url.JoinPath(origin, "index.html")
	if err != nil {
		return err
	}
	target := objc.ID(objc.GetClass("NSURL")).Send(selURLWithString, nsString(page))
	request := objc.ID(objc.GetClass("NSURLRequest")).Send(selRequestWithURL, target)
	webView.Send(selLoadRequest, request)
	return nil
}

func mainScreenBounds() nsRect {
	screen := objc.ID(objc.GetClass("UIScreen")).Send(selMainScreen)
	if screen == 0 {
		return nsRect{}
	}
	return objectBounds(screen)
}

func objectBounds(object objc.ID) nsRect {
	if object == 0 {
		return nsRect{}
	}
	if boundsMsg == nil {
		native.Register(&boundsMsg, native.MsgSend())
	}
	return boundsMsg(object, selBounds)
}

func pumpIOS() {
	if !thread.On() {
		return
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selPoolNew)
	defer pool.Send(selPoolDrain)
	loop := objc.ID(objc.GetClass("NSRunLoop")).Send(selCurrentRunLoop)
	if loop != 0 {
		// distantPast returns as soon as the queue is empty, then thread.Do runs.
		// The mode name is kCFRunLoopDefaultMode, the same string AppKit polls with.
		when := objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
		loop.Send(selRunUntil, nsString("kCFRunLoopDefaultMode"), when)
	}
	// A zero wait skips the before-waiting observers, so the white window never commits.
	flushDisplay()
}

func flushDisplay() {
	displayOnce.Do(func() {
		_, _ = native.Open("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", native.Global|native.Lazy)
	})
	if class := objc.GetClass("CATransaction"); class != 0 {
		objc.ID(class).Send(selFlush)
	}
}

func (view *webKitView) useSchemeIOS(scheme daynight.Mode, flip bool) {
	thread.Do(func() {
		if view == nil || view.window == 0 {
			return
		}
		style := styleLight
		other := styleDark
		if scheme == daynight.Dark {
			style = styleDark
			other = styleLight
		}
		if flip && view.webView != 0 {
			view.webView.Send(selOverrideStyle, other)
		}
		view.window.Send(selOverrideStyle, style)
		if view.webView != 0 {
			view.webView.Send(selOverrideStyle, style)
		}
	})
}

func (view *webKitView) closeIOS() error {
	thread.Do(func() {
		if view.webView != 0 {
			view.webView.Send(selRemoveSubview)
		}
		if view.window != 0 {
			view.window.Send(selSetHidden, true)
		}
		view.markClosed()
	})
	return nil
}
