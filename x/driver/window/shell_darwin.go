//go:build darwin

package window

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	cocoaFramework    = "/System/Library/Frameworks/Cocoa.framework/Cocoa"
	appServicesPath   = "/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices"
	policyRegular     = 0
	transformStateFG  = 1
	currentProcessPSN = 2
)

// processSerial is ProcessSerialNumber. Both fields are UInt32.
type processSerial struct {
	high uint32
	low  uint32
}

var shellReady atomic.Bool

func showShell(title string, icon image.Image) {
	// An off-main call must not stick, or the real UI thread would skip adoption.
	if !thread.ProcessMain() {
		return
	}
	if shellReady.Load() {
		return
	}
	if !adoptShell(title, icon) {
		return
	}
	shellReady.Store(true)
}

func adoptShell(title string, icon image.Image) bool {
	// The packaged helper lives in Contents/MacOS and must not take a Dock tile.
	// The parent .app icon comes from the asset catalog.
	exe, err := os.Executable()
	if err == nil && strings.Contains(filepath.ToSlash(exe), ".app/Contents/MacOS/") {
		return true
	}
	// Activation policy alone leaves a bare executable out of the Dock on older macOS.
	// Once NSApplication exists, TransformProcessType returns paramErr; still set the icon.
	transformForeground()
	if _, err := native.Open(cocoaFramework, native.Global|native.Lazy); err != nil {
		return false
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(objc.RegisterName("new"))
	defer pool.Send(objc.RegisterName("drain"))
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	if app == 0 {
		return false
	}
	app.Send(objc.RegisterName("setActivationPolicy:"), policyRegular)
	if app.Send(objc.RegisterName("isRunning")) == 0 {
		app.Send(objc.RegisterName("finishLaunching"))
	}
	setAppIcon(app, icon)
	ensureMenu(app, title)
	return true
}

func transformForeground() bool {
	lib, err := native.Open(appServicesPath, native.Global|native.Lazy)
	if err != nil {
		return false
	}
	var fn func(*processSerial, uint32) int32
	if err := native.Bind(lib, "TransformProcessType", &fn); err != nil || fn == nil {
		return false
	}
	psn := processSerial{low: currentProcessPSN}
	return fn(&psn, transformStateFG) == 0
}

func setAppIcon(app objc.ID, icon image.Image) {
	if icon == nil {
		return
	}
	png, err := icons.EncodePNG(icon)
	if err != nil || len(png) == 0 {
		return
	}
	data := objc.ID(objc.GetClass("NSData")).Send(
		objc.RegisterName("dataWithBytes:length:"),
		unsafe.Pointer(&png[0]), len(png),
	)
	if data == 0 {
		return
	}
	img := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(
		objc.RegisterName("initWithData:"), data,
	)
	if img == 0 {
		return
	}
	app.Send(objc.RegisterName("setApplicationIconImage:"), img)
}

func ensureMenu(app objc.ID, title string) {
	if app.Send(objc.RegisterName("mainMenu")) != 0 {
		return
	}
	name := menuName(title)
	bar := objc.ID(objc.GetClass("NSMenu")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	item := objc.ID(objc.GetClass("NSMenuItem")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("init"))
	bar.Send(objc.RegisterName("addItem:"), item)
	menu := objc.ID(objc.GetClass("NSMenu")).Send(objc.RegisterName("alloc")).Send(
		objc.RegisterName("initWithTitle:"), nsString(name),
	)
	menu.Send(objc.RegisterName("setAutoenablesItems:"), false)
	quit := objc.ID(objc.GetClass("NSMenuItem")).Send(objc.RegisterName("alloc")).Send(
		objc.RegisterName("initWithTitle:action:keyEquivalent:"),
		nsString("Quit "+name),
		objc.RegisterName("terminate:"),
		nsString("q"),
	)
	quit.Send(objc.RegisterName("setTarget:"), app)
	menu.Send(objc.RegisterName("addItem:"), quit)
	item.Send(objc.RegisterName("setSubmenu:"), menu)
	app.Send(objc.RegisterName("setMainMenu:"), bar)
}

func menuName(title string) string {
	bundle := objc.ID(objc.GetClass("NSBundle")).Send(objc.RegisterName("mainBundle"))
	if name := infoString(bundle, "CFBundleDisplayName"); name != "" {
		return name
	}
	if name := infoString(bundle, "CFBundleName"); name != "" {
		return name
	}
	if strings.TrimSpace(title) != "" {
		return title
	}
	exe, err := os.Executable()
	if err != nil {
		return "App"
	}
	return filepath.Base(exe)
}

func infoString(bundle objc.ID, key string) string {
	if bundle == 0 {
		return ""
	}
	return goString(bundle.Send(objc.RegisterName("objectForInfoDictionaryKey:"), nsString(key)))
}

func nsString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("stringWithUTF8String:"), unsafe.Pointer(&raw[0]))
}

func goString(object objc.ID) string {
	if object == 0 {
		return ""
	}
	pointer := object.Send(objc.RegisterName("UTF8String"))
	if pointer == 0 {
		return ""
	}
	p := uintptr(pointer)
	n := 0
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
		if n > 4096 {
			break
		}
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}
