//go:build windows

package webview2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	_ "github.com/lewtec/lewkit/x/driver/messagebox/win32"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
	webview2 "github.com/lewtec/lewkit/x/ffi/native/webview2"
	lewrelease "github.com/lewtec/lewkit/x/release"
)

const (
	wmClose        = 0x0010
	wmDestroy      = 0x0002
	wmMove         = 0x0003
	wmSize         = 0x0005
	wmSysCommand   = 0x0112
	scClose        = 0xF060
	wmJob          = 0x8000 + 1
	wsOverlapped   = 0x00CF0000
	wsClipChildren = 0x02000000
	wsExAppWindow  = 0x00040000
	swShow         = 5
	cwUseDefault   = 0x80000000
	swpNoSize      = 0x0001
	swpNoMove      = 0x0002
	swpShowWindow  = 0x0040
	pmRemove       = 0x0001
	colorWindow    = 5
)

var (
	classOnce sync.Once
	classErr  error
	classAtom uintptr

	loopOnce       sync.Once
	loopStarted    = make(chan struct{})
	loopErr        error
	loopJobs       = make(chan func(), 32)
	loopWindow     uintptr
	nextIdentifier atomic.Uint64
	windows        sync.Map

	procGetForegroundWindow      = native.ProcOf("user32.dll", "GetForegroundWindow")
	procGetWindowThreadProcessId = native.ProcOf("user32.dll", "GetWindowThreadProcessId")
	procSetForegroundWindow      = native.ProcOf("user32.dll", "SetForegroundWindow")
	procBringWindowToTop         = native.ProcOf("user32.dll", "BringWindowToTop")
	procSetWindowPos             = native.ProcOf("user32.dll", "SetWindowPos")
	procAttachThreadInput        = native.ProcOf("user32.dll", "AttachThreadInput")
	procAllowSetForegroundWindow = native.ProcOf("user32.dll", "AllowSetForegroundWindow")
	procGetCurrentProcessId      = native.ProcOf("kernel32.dll", "GetCurrentProcessId")
	procGetCurrentThreadId       = native.ProcOf("kernel32.dll", "GetCurrentThreadId")

	environmentIID        = guid{0xB96D755E, 0x0319, 0x4E92, [8]byte{0xA2, 0x96, 0x23, 0x43, 0x6F, 0x46, 0xA1, 0xFC}}
	controllerIID         = guid{0x4D00C0D1, 0x9434, 0x4EB6, [8]byte{0x80, 0x78, 0x86, 0x97, 0xA5, 0x60, 0x33, 0x4F}}
	controller2IID        = guid{0xC979903E, 0xD4CA, 0x4228, [8]byte{0x92, 0xEB, 0x47, 0xEE, 0x3F, 0xA9, 0x6E, 0xAB}}
	webViewIID            = guid{0x76ECEACB, 0x0462, 0x4D94, [8]byte{0xAC, 0x83, 0x42, 0x3A, 0x67, 0x93, 0x77, 0x5E}}
	webView13IID          = guid{0xF75F09A8, 0x667E, 0x4983, [8]byte{0x88, 0xD6, 0xC8, 0x77, 0x3F, 0x31, 0x5E, 0x84}}
	unknownIID            = guid{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	environmentHandlerIID = guid{0x4E8A3389, 0xC9D8, 0x4BD2, [8]byte{0xB6, 0xB5, 0x12, 0x4F, 0xEE, 0x6C, 0xC1, 0x4D}}
	controllerHandlerIID  = guid{0x6C4819F3, 0xC9B7, 0x4260, [8]byte{0x81, 0x27, 0xC9, 0xF5, 0xBD, 0xE7, 0xF6, 0x8C}}
	messageHandlerIID     = guid{0x57213F19, 0x00E6, 0x49FA, [8]byte{0x8E, 0x07, 0x89, 0x8E, 0xA0, 0x1E, 0xCB, 0xD2}}
	resourceHandlerIID    = guid{0xAB00B74C, 0x15F1, 0x4646, [8]byte{0x80, 0xE8, 0xE7, 0x63, 0x41, 0xD2, 0x5D, 0x71}}
	scriptHandlerIID      = guid{0x49511172, 0xCC67, 0x4BCA, [8]byte{0x99, 0x23, 0x13, 0x71, 0x12, 0xF4, 0xC4, 0xCC}}

	queryCallback             = syscall.NewCallback(queryInterface)
	addRefCallback            = syscall.NewCallback(addReference)
	releaseCallback           = syscall.NewCallback(releaseInterface)
	environmentInvokeCallback = syscall.NewCallback(environmentInvoke)
	controllerInvokeCallback  = syscall.NewCallback(controllerInvoke)
	messageInvokeCallback     = syscall.NewCallback(messageInvoke)
	resourceInvokeCallback    = syscall.NewCallback(resourceInvoke)
	scriptInvokeCallback      = syscall.NewCallback(scriptInvoke)
	windowProcedureCallback   = syscall.NewCallback(windowProcedure)
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type handlerVtbl struct {
	query, addRef, release, invoke uintptr
}

type comHandler struct {
	vtbl   *handlerVtbl
	iid    guid
	ref    int32
	done   chan uintptr
	failed chan error
	view   *edgeView
	call   *evaluation
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menu       *uint16
	class      *uint16
	iconSmall  uintptr
}

type message struct {
	hwnd    uintptr
	message uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
}

func loader() error { return webview2.Available() }

func (edgeDriver) Open(ctx context.Context, cfg webview.Config) (webview.View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := ensureLoop(); err != nil {
		return nil, err
	}
	view := &edgeView{
		ctx:        ctx,
		identifier: uintptr(nextIdentifier.Add(1)),
		html:       cfg.HTML,
		files:      cfg.FS,
		handler:    cfg.Handler,
		title:      cfg.Title,
		icon:       cfg.Icon,
		profile:    cfg.Profile,
		messages:   make(chan []byte, 32),
		done:       make(chan struct{}),
	}
	width, height, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	view.width = int32(width)
	view.height = int32(height)
	out := make(chan error, 1)
	post(func() {
		defer func() {
			if rec := recover(); rec != nil {
				out <- fmt.Errorf("webview: %v", rec)
			}
		}()
		out <- view.create(ctx)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-out:
		if err != nil {
			return nil, err
		}
	}
	context.AfterFunc(ctx, func() { _ = view.Close() })
	webview.Follow(ctx, func(scheme daynight.Mode) {
		post(func() { view.useScheme(scheme) })
	})
	return view, nil
}

func ensureLoop() error {
	loopOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			if err := webview2.CoInitialize(); err != nil {
				loopErr = err
				close(loopStarted)
				return
			}
			if err := registerWindowClass(); err != nil {
				loopErr = err
				close(loopStarted)
				return
			}
			// HWND_MESSAGE is a message-only window. It keeps the queue alive.
			hwndMessage := ^uintptr(2)
			hwnd, _, _ := webview2.CreateWindowEx.Call(0, classAtom, 0, 0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
			if hwnd == 0 {
				loopErr = fmt.Errorf("%w: message window", driver.ErrUnavailable)
				close(loopStarted)
				return
			}
			loopWindow = hwnd
			close(loopStarted)
			var msg message
			for {
				ret, _, _ := webview2.GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
				if int32(ret) <= 0 {
					return
				}
				// Run the job on this thread, outside the window procedure.
				// A file dialog posted to this window has to start here too.
				// Show inside the window procedure returns E_FAIL.
				if msg.message == wmJob {
					drainJobs()
					continue
				}
				if window.DeliverUI(uintptr(msg.message), msg.wparam) {
					continue
				}
				_, _, _ = webview2.TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
				_, _, _ = webview2.DispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
			}
		}()
	})
	<-loopStarted
	return loopErr
}

func registerWindowClass() error {
	classOnce.Do(func() {
		instance, _, _ := webview2.GetModuleHandle.Call(0)
		name, err := syscall.UTF16PtrFromString(lewrelease.Name() + ".webview")
		if err != nil {
			classErr = err
			return
		}
		class := wndClassEx{
			wndProc:    windowProcedureCallback,
			instance:   instance,
			background: colorWindow + 1,
			class:      name,
		}
		class.size = uint32(unsafe.Sizeof(class))
		atom, _, err := webview2.RegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))
		if atom == 0 {
			classErr = fmt.Errorf("register class: %v", err)
			return
		}
		classAtom = atom
	})
	return classErr
}

func post(job func()) {
	loopJobs <- job
	if loopWindow != 0 {
		_, _, _ = webview2.PostMessage.Call(loopWindow, wmJob, 0, 0)
	}
}

func drainJobs() {
	for {
		select {
		case job := <-loopJobs:
			job()
		default:
			return
		}
	}
}

type edgeView struct {
	ctx         context.Context
	identifier  uintptr
	html        string
	files       fs.FS
	handler     http.Handler
	title       string
	icon        image.Image
	profile     string
	width       int32
	height      int32
	messages    chan []byte
	done        chan struct{}
	closed      atomic.Bool
	ready       atomic.Bool
	fail        atomic.Value // error
	held        []*comHandler
	hwnd        uintptr
	controller  uintptr
	webView     uintptr
	environment uintptr
}

func (view *edgeView) hold(handler *comHandler) {
	if handler == nil {
		return
	}
	handler.view = view
	view.held = append(view.held, handler)
}

func (view *edgeView) noteFail(err error) {
	if view == nil || err == nil {
		return
	}
	if !view.fail.CompareAndSwap(nil, err) {
		return
	}
	if view.ready.Load() {
		view.report(err)
	}
}

func (view *edgeView) failure() error {
	err, _ := view.fail.Load().(error)
	return err
}

func (view *edgeView) create(ctx context.Context) error {
	title, err := syscall.UTF16PtrFromString(view.title)
	if err != nil {
		return err
	}
	instance, _, _ := webview2.GetModuleHandle.Call(0)
	owner := foregroundProcessWindow()
	exStyle := uintptr(0)
	if owner != 0 {
		exStyle = wsExAppWindow
	}
	hwnd, _, _ := webview2.CreateWindowEx.Call(exStyle, classAtom, uintptr(unsafe.Pointer(title)), wsOverlapped|wsClipChildren, cwUseDefault, cwUseDefault, uintptr(view.width), uintptr(view.height), owner, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("%w: window", driver.ErrUnavailable)
	}
	view.hwnd = hwnd
	windows.Store(hwnd, view)
	window.RegisterUI(hwnd)
	window.ApplyWindowIcon(hwnd, view.icon)
	defer func() {
		if !view.ready.Load() {
			view.closeWindow()
		}
	}()
	var folderUTF *uint16
	profile := strings.TrimSpace(view.profile)
	if profile != "" {
		if err := os.MkdirAll(profile, 0o755); err != nil {
			return err
		}
		folderUTF, err = syscall.UTF16PtrFromString(profile)
		if err != nil {
			return err
		}
	}
	environmentHandler := newHandler(environmentHandlerIID, environmentInvokeCallback)
	environmentHandler.done = make(chan uintptr, 1)
	environmentHandler.failed = make(chan error, 1)
	view.hold(environmentHandler)
	if err := webview2.CreateEnvironment(folderUTF, uintptr(unsafe.Pointer(environmentHandler))); err != nil {
		return err
	}
	environment, err := waitFor(ctx, environmentHandler.done, environmentHandler.failed, view.pumpStartup)
	runtime.KeepAlive(environmentHandler)
	if err != nil {
		return err
	}
	view.environment = environment
	controllerHandler := newHandler(controllerHandlerIID, controllerInvokeCallback)
	controllerHandler.done = make(chan uintptr, 1)
	controllerHandler.failed = make(chan error, 1)
	view.hold(controllerHandler)
	hr := call(view.environment, 3, hwnd, uintptr(unsafe.Pointer(controllerHandler)))
	if hr < 0 {
		return fmt.Errorf("CreateCoreWebView2Controller: %x", uint32(hr))
	}
	controller, err := waitFor(ctx, controllerHandler.done, controllerHandler.failed, view.pumpStartup)
	runtime.KeepAlive(controllerHandler)
	if err != nil {
		return err
	}
	view.controller = controller
	var web uintptr
	hr = call(view.controller, 25, uintptr(unsafe.Pointer(&web)))
	if hr < 0 || web == 0 {
		return fmt.Errorf("get_CoreWebView2: %x", uint32(hr))
	}
	view.webView, err = query(web, webViewIID)
	release(web)
	if err != nil {
		return err
	}
	view.resize()
	view.useBackground()
	_ = call(view.controller, 4, 1) // put_IsVisible TRUE
	messageHandler := newHandler(messageHandlerIID, messageInvokeCallback)
	view.hold(messageHandler)
	var token int64
	hr = call(view.webView, 34, uintptr(unsafe.Pointer(messageHandler)), uintptr(unsafe.Pointer(&token)))
	if hr < 0 {
		return fmt.Errorf("add_WebMessageReceived: %x", uint32(hr))
	}
	resourceHandler := newHandler(resourceHandlerIID, resourceInvokeCallback)
	view.hold(resourceHandler)
	hr = call(view.webView, 55, uintptr(unsafe.Pointer(resourceHandler)), uintptr(unsafe.Pointer(&token)))
	if hr < 0 {
		return fmt.Errorf("add_WebResourceRequested: %x", uint32(hr))
	}
	filter, err := syscall.UTF16PtrFromString("https://view*" + webview.HostSuffix() + "/*")
	if err != nil {
		return err
	}
	hr = call(view.webView, 57, uintptr(unsafe.Pointer(filter)), 0)
	if hr < 0 {
		return fmt.Errorf("AddWebResourceRequestedFilter: %x", uint32(hr))
	}
	if scheme, err := daynight.Current(ctx); err == nil {
		view.useScheme(scheme)
	}
	target := fmt.Sprintf("https://view%d%s/index.html", view.identifier, webview.HostSuffix())
	if view.handler != nil && strings.TrimSpace(view.html) == "" {
		target = fmt.Sprintf("https://view%d%s/", view.identifier, webview.HostSuffix())
	}
	targetUTF, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	hr = call(view.webView, 5, uintptr(unsafe.Pointer(targetUTF)))
	if hr < 0 {
		return fmt.Errorf("Navigate: %x", uint32(hr))
	}
	if err := view.startupErr(); err != nil {
		return err
	}
	_, _, _ = webview2.ShowWindow.Call(hwnd, swShow)
	raiseWindow(hwnd)
	if err := view.startupErr(); err != nil {
		return err
	}
	view.ready.Store(true)
	return nil
}

func foregroundProcessWindow() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	current, _, _ := procGetCurrentProcessId.Call()
	if uint32(current) != pid {
		return 0
	}
	return hwnd
}

func raiseWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	procAllowSetForegroundWindow.Call(uintptr(0xFFFFFFFF))
	fg, _, _ := procGetForegroundWindow.Call()
	var fgThread uintptr
	if fg != 0 {
		fgThread, _, _ = procGetWindowThreadProcessId.Call(fg, 0)
	}
	our, _, _ := procGetCurrentThreadId.Call()
	attached := false
	if fgThread != 0 && fgThread != our {
		r, _, _ := procAttachThreadInput.Call(our, fgThread, 1)
		attached = r != 0
	}
	procSetForegroundWindow.Call(hwnd)
	procBringWindowToTop.Call(hwnd)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	if attached {
		procAttachThreadInput.Call(our, fgThread, 0)
	}
}

func (view *edgeView) startupErr() error {
	if err := view.failure(); err != nil {
		return err
	}
	select {
	case <-view.done:
		return errClosed
	default:
		return nil
	}
}

// errClosed means the host window was destroyed before the page was shown.
var errClosed = errors.New("webview closed before the page was ready")

// pumpStartup stops waiting once the window has closed.
func (view *edgeView) pumpStartup(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := view.startupErr(); err != nil {
		return err
	}
	if err := pumpWaiting(ctx); err != nil {
		return err
	}
	return view.startupErr()
}

// pumpWaiting dispatches one queued message.
// An empty queue sleeps. MsgWaitForMultipleObjects reports a wake for queue
// state PeekMessage cannot remove, and that loop pegs a core after close.
func pumpWaiting(context.Context) error {
	var msg message
	ret, _, _ := webview2.PeekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmRemove)
	if ret == 0 {
		time.Sleep(15 * time.Millisecond)
		return nil
	}
	if msg.message == wmJob {
		drainJobs()
		return nil
	}
	if window.DeliverUI(uintptr(msg.message), msg.wparam) {
		return nil
	}
	_, _, _ = webview2.TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
	_, _, _ = webview2.DispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	return nil
}

func (view *edgeView) closeWindow() {
	hwnd := view.hwnd
	if hwnd != 0 {
		_, _, _ = webview2.DestroyWindow.Call(hwnd)
	}
	view.markClosed()
}

func (view *edgeView) useBackground() {
	if view.controller == 0 {
		return
	}
	newer, err := query(view.controller, controller2IID)
	if err != nil {
		return
	}
	defer release(newer)
	// ICoreWebView2Controller2.put_DefaultBackgroundColor. Opaque white.
	_ = call(newer, 27, 0xFFFFFFFF)
}

func (view *edgeView) moved() {
	if view.controller == 0 {
		return
	}
	_ = call(view.controller, 23) // NotifyParentWindowPositionChanged
}

// useScheme sets ICoreWebView2Profile.PreferredColorScheme. That updates
// prefers-color-scheme on the loaded page without a navigation.
// 0 is auto, 1 is light, 2 is dark.
func (view *edgeView) useScheme(scheme daynight.Mode) {
	if view == nil || view.webView == 0 {
		return
	}
	newer, err := query(view.webView, webView13IID)
	if err != nil {
		return
	}
	defer release(newer)
	var profile uintptr
	if call(newer, 105, uintptr(unsafe.Pointer(&profile))) < 0 || profile == 0 {
		return
	}
	defer release(profile)
	value := uintptr(1)
	if scheme == daynight.Dark {
		value = 2
	}
	_ = call(profile, 9, value)
}

func (view *edgeView) resize() {
	if view.controller == 0 || view.hwnd == 0 {
		return
	}
	var bounds rect
	_, _, _ = webview2.GetClientRect.Call(view.hwnd, uintptr(unsafe.Pointer(&bounds)))
	_ = call(view.controller, 6, uintptr(unsafe.Pointer(&bounds)))
}

func (view *edgeView) Messages() <-chan []byte { return view.messages }
func (view *edgeView) Done() <-chan struct{}   { return view.done }

func (view *edgeView) Close() error {
	post(func() {
		if view.hwnd != 0 {
			_, _, _ = webview2.PostMessage.Call(view.hwnd, wmClose, 0, 0)
			return
		}
		view.markClosed()
	})
	return nil
}

func (view *edgeView) markClosed() {
	if !view.closed.CompareAndSwap(false, true) {
		return
	}
	windows.Delete(view.hwnd)
	window.ForgetUI(view.hwnd)
	controller := view.controller
	web := view.webView
	env := view.environment
	view.hwnd = 0
	view.controller = 0
	view.webView = 0
	view.environment = 0
	if controller != 0 {
		_ = call(controller, 24) // Close. A nested window message returns here.
		release(controller)
	}
	if web != 0 {
		release(web)
	}
	if env != 0 {
		release(env)
	}
	view.held = nil
	close(view.done)
}

func (view *edgeView) Evaluate(ctx context.Context, script string) (string, error) {
	select {
	case <-view.done:
		return "", webview.ErrClosed
	default:
	}
	call := &evaluation{done: make(chan struct{})}
	defer runtime.KeepAlive(call)
	post(func() {
		if view.webView == 0 {
			call.err = webview.ErrClosed
			close(call.done)
			return
		}
		scriptUTF, err := syscall.UTF16PtrFromString(script)
		if err != nil {
			call.err = err
			close(call.done)
			return
		}
		handler := newHandler(scriptHandlerIID, scriptInvokeCallback)
		handler.call = call
		handler.view = view
		call.handler = handler
		hr := callCOM(view.webView, 29, uintptr(unsafe.Pointer(scriptUTF)), uintptr(unsafe.Pointer(handler)))
		if hr < 0 {
			call.err = fmt.Errorf("ExecuteScript: %x", uint32(hr))
			close(call.done)
		}
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-view.done:
		return "", webview.ErrClosed
	case <-call.done:
		return call.text, call.err
	}
}

type evaluation struct {
	text    string
	err     error
	done    chan struct{}
	handler *comHandler
}

func newHandler(iid guid, invoke uintptr) *comHandler {
	handler := &comHandler{iid: iid, ref: 1}
	handler.vtbl = &handlerVtbl{
		query:   queryCallback,
		addRef:  addRefCallback,
		release: releaseCallback,
		invoke:  invoke,
	}
	return handler
}

func queryInterface(this, iidPointer, out uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	requested := *(*guid)(unsafe.Pointer(iidPointer))
	if requested == unknownIID || requested == handler.iid {
		*(*uintptr)(unsafe.Pointer(out)) = this
		atomic.AddInt32(&handler.ref, 1)
		return 0
	}
	*(*uintptr)(unsafe.Pointer(out)) = 0
	return 0x80004002
}

func addReference(this uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&handler.ref, 1))
}

func releaseInterface(this uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&handler.ref, -1))
}

func environmentInvoke(this, result, environment uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	defer handler.catch()
	if int32(result) < 0 || environment == 0 {
		handler.fail(fmt.Errorf("environment: %x", uint32(result)))
		return 0
	}
	queried, err := query(environment, environmentIID)
	if err != nil {
		handler.fail(err)
		return 0
	}
	handler.done <- queried
	return 0
}

func controllerInvoke(this, result, controller uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	defer handler.catch()
	if int32(result) < 0 || controller == 0 {
		handler.fail(fmt.Errorf("controller: %x", uint32(result)))
		return 0
	}
	queried, err := query(controller, controllerIID)
	if err != nil {
		handler.fail(err)
		return 0
	}
	handler.done <- queried
	return 0
}

func messageInvoke(this, _, args uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	defer handler.catch()
	if handler.view == nil || args == 0 {
		return 0
	}
	var textPointer uintptr
	hr := call(args, 4, uintptr(unsafe.Pointer(&textPointer)))
	if hr < 0 || textPointer == 0 {
		return 0
	}
	text := utf16Ptr(textPointer)
	webview2.FreeTaskMemory(textPointer)
	payload := []byte(text)
	select {
	case handler.view.messages <- payload:
	default:
		go func() {
			select {
			case handler.view.messages <- payload:
			case <-handler.view.done:
			}
		}()
	}
	return 0
}

func resourceInvoke(this, _, args uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	defer handler.catch()
	if handler.view == nil || args == 0 {
		return 0
	}
	var request uintptr
	if call(args, 3, uintptr(unsafe.Pointer(&request))) < 0 || request == 0 {
		return 0
	}
	defer release(request)
	var uriPointer uintptr
	if call(request, 3, uintptr(unsafe.Pointer(&uriPointer))) < 0 || uriPointer == 0 {
		return 0
	}
	raw := utf16Ptr(uriPointer)
	webview2.FreeTaskMemory(uriPointer)
	parsed, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	if handler.view.handler != nil {
		method := "GET"
		var methodPointer uintptr
		// ICoreWebView2WebResourceRequest: 5 GetMethod, 7 GetContent.
		if call(request, 5, uintptr(unsafe.Pointer(&methodPointer))) >= 0 && methodPointer != 0 {
			method = utf16Ptr(methodPointer)
			webview2.FreeTaskMemory(methodPointer)
		}
		bodyBytes := []byte{}
		var content uintptr
		if call(request, 7, uintptr(unsafe.Pointer(&content))) >= 0 && content != 0 {
			stream := &comStream{stream: content}
			read, readErr := io.ReadAll(stream)
			_ = stream.Close()
			if readErr != nil {
				return 0
			}
			bodyBytes = read
		}
		// ICoreWebView2WebResourceRequestedEventArgs slot 6 is GetDeferral.
		// The page handler, including a GUI window, runs off this thread.
		var deferral uintptr
		if call(args, 6, uintptr(unsafe.Pointer(&deferral))) < 0 || deferral == 0 {
			status, responseHeader, payload, err := dispatchResource(handler.view, method, raw, parsed.Path, bodyBytes)
			if err != nil {
				if payload != nil {
					payload.Close()
				}
				return 0
			}
			writeWebResource(handler.view, args, status, responseHeader, payload)
			return 0
		}
		_ = call(args, 1)
		view := handler.view
		go func() {
			status, responseHeader, payload, err := dispatchResource(view, method, raw, parsed.Path, bodyBytes)
			post(func() {
				if err == nil && view.environment != 0 {
					writeWebResource(view, args, status, responseHeader, payload)
				} else if payload != nil {
					payload.Close()
				}
				completeDeferral(deferral)
				release(args)
			})
		}()
		return 0
	}
	body, contentType, err := webview.ReadPage(handler.view.html, handler.view.files, parsed.Path)
	if err != nil {
		return 0
	}
	if parsed.Path == "/" || parsed.Path == "/index.html" {
		body = append([]byte("<script>"+webview.ChromeBridge()+"</script>"), body...)
	}
	stream, err := webview2.MemoryStream(body)
	if err != nil {
		return 0
	}
	defer release(stream)
	reason, _ := syscall.UTF16PtrFromString("OK")
	headers, _ := syscall.UTF16PtrFromString("Content-Type: " + contentType + "\r\n")
	var response uintptr
	hr := call(handler.view.environment, 4, stream, 200, uintptr(unsafe.Pointer(reason)), uintptr(unsafe.Pointer(headers)), uintptr(unsafe.Pointer(&response)))
	if hr < 0 || response == 0 {
		return 0
	}
	defer release(response)
	_ = call(args, 5, response)
	return 0
}

func scriptInvoke(this, result, jsonPointer uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	defer handler.catch()
	if handler.call == nil {
		return 0
	}
	if int32(result) < 0 {
		handler.call.err = fmt.Errorf("script: %x", uint32(result))
	} else if jsonPointer != 0 {
		handler.call.text = utf16Ptr(jsonPointer)
		webview2.FreeTaskMemory(jsonPointer)
	}
	close(handler.call.done)
	return 0
}

func windowProcedure(hwnd, msg, wparam, lparam uintptr) uintptr {
	defer func() {
		if rec := recover(); rec != nil {
			if loaded, ok := windows.Load(hwnd); ok {
				loaded.(*edgeView).noteFail(fmt.Errorf("webview: %v", rec))
			} else {
				slog.Error("webview", "err", rec)
			}
		}
	}()
	if window.DeliverUI(msg, wparam) {
		return 1
	}
	if msg == wmJob {
		drainJobs()
		return 0
	}
	loaded, ok := windows.Load(hwnd)
	if !ok {
		ret, _, _ := webview2.DefWindowProc.Call(hwnd, msg, wparam, lparam)
		return ret
	}
	view := loaded.(*edgeView)
	switch msg {
	case wmSize:
		view.resize()
	case wmMove:
		view.moved()
	case wmSysCommand:
		if !view.ready.Load() && wparam&0xFFF0 == scClose {
			return 0
		}
	case wmClose:
		if !view.ready.Load() {
			return 0
		}
		if view.hwnd != 0 {
			_, _, _ = webview2.DestroyWindow.Call(view.hwnd)
		} else {
			view.markClosed()
		}
		return 0
	case wmDestroy:
		view.markClosed()
	}
	ret, _, _ := webview2.DefWindowProc.Call(hwnd, msg, wparam, lparam)
	return ret
}

func (handler *comHandler) fail(err error) {
	if handler == nil || err == nil {
		return
	}
	if handler.view != nil {
		handler.view.noteFail(err)
	}
	if handler.failed == nil {
		return
	}
	select {
	case handler.failed <- err:
	default:
	}
}

func (handler *comHandler) catch() {
	rec := recover()
	if rec == nil || handler == nil {
		return
	}
	err := fmt.Errorf("webview: %v", rec)
	if handler.call != nil {
		handler.call.err = err
		select {
		case <-handler.call.done:
		default:
			close(handler.call.done)
		}
	}
	handler.fail(err)
}

var reported sync.Once

func (view *edgeView) report(err error) {
	if view == nil || err == nil || view.ctx == nil {
		return
	}
	ctx := view.ctx
	reported.Do(func() {
		_ = messagebox.Show(ctx, lewrelease.Name(), err.Error())
	})
}

func method(object uintptr, index int) uintptr {
	vtable := *(*uintptr)(unsafe.Pointer(object))
	return *(*uintptr)(unsafe.Pointer(vtable + uintptr(index)*unsafe.Sizeof(uintptr(0))))
}

func call(object uintptr, index int, args ...uintptr) int32 {
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, object)
	all = append(all, args...)
	hr, _, _ := syscall.SyscallN(method(object, index), all...)
	return int32(hr)
}

func callCOM(object uintptr, index int, args ...uintptr) int32 { return call(object, index, args...) }

func query(object uintptr, iid guid) (uintptr, error) {
	var out uintptr
	hr := call(object, 0, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&out)))
	if hr < 0 || out == 0 {
		return 0, fmt.Errorf("QueryInterface: %x", uint32(hr))
	}
	return out, nil
}

func release(object uintptr) {
	if object != 0 {
		_ = call(object, 2)
	}
}

func utf16Ptr(pointer uintptr) string {
	if pointer == 0 {
		return ""
	}
	var units []uint16
	for {
		unit := *(*uint16)(unsafe.Pointer(pointer + uintptr(len(units)*2)))
		if unit == 0 {
			break
		}
		units = append(units, unit)
		if len(units) > 1<<20 {
			break
		}
	}
	return syscall.UTF16ToString(units)
}

func dispatchResource(view *edgeView, method, raw, path string, body []byte) (status int, header http.Header, payload io.ReadCloser, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("webview: %v", rec)
		}
	}()
	status, header, payload, err = webview.Dispatch(view.handler, method, raw, nil, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return status, header, payload, err
	}
	if strings.Contains(header.Get("Content-Type"), "html") && (path == "/" || path == "/index.html") {
		payload = prefixReader(payload, "<script>"+webview.ChromeBridge()+"</script>")
	}
	return status, header, payload, nil
}

func completeDeferral(deferral uintptr) {
	if deferral == 0 {
		return
	}
	// ICoreWebView2Deferral slot 3 is Complete.
	_ = call(deferral, 3)
	release(deferral)
}

func writeWebResource(view *edgeView, args uintptr, status int, header http.Header, body io.ReadCloser) {
	stream := newByteStream(body)
	reasonText := http.StatusText(status)
	if reasonText == "" {
		reasonText = "OK"
	}
	reason, _ := syscall.UTF16PtrFromString(reasonText)
	headerText := webview.HeaderText(header)
	headers, _ := syscall.UTF16PtrFromString(headerText)
	var response uintptr
	hr := call(view.environment, 4, stream, uintptr(status), uintptr(unsafe.Pointer(reason)), uintptr(unsafe.Pointer(headers)), uintptr(unsafe.Pointer(&response)))
	if hr < 0 || response == 0 {
		release(stream)
		return
	}
	defer release(response)
	_ = call(args, 5, response)
}

type comStream struct {
	stream uintptr
}

func (stream *comStream) Read(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}
	var count uint32
	hr := call(stream.stream, 3, uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), uintptr(unsafe.Pointer(&count)))
	if hr < 0 {
		return 0, fmt.Errorf("stream read: %x", uint32(hr))
	}
	if count == 0 {
		return 0, io.EOF
	}
	return int(count), nil
}

func (stream *comStream) Close() error {
	release(stream.stream)
	return nil
}

type prefixReadCloser struct {
	prefix []byte
	rest   io.ReadCloser
}

func prefixReader(rest io.ReadCloser, prefix string) io.ReadCloser {
	return &prefixReadCloser{prefix: []byte(prefix), rest: rest}
}

func (reader *prefixReadCloser) Read(payload []byte) (int, error) {
	if len(reader.prefix) > 0 {
		count := copy(payload, reader.prefix)
		reader.prefix = reader.prefix[count:]
		return count, nil
	}
	return reader.rest.Read(payload)
}

func (reader *prefixReadCloser) Close() error { return reader.rest.Close() }

var (
	streamIID             = guid{0x0000000C, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	liveStreams           sync.Map
	streamNotImplemented  = syscall.NewCallback(func(uintptr) uintptr { return 0x80004001 })
	streamQueryCallback   = syscall.NewCallback(streamQuery)
	streamAddRefCallback  = syscall.NewCallback(streamAddRef)
	streamReleaseCallback = syscall.NewCallback(streamRelease)
	streamReadCallback    = syscall.NewCallback(streamReadBytes)
)

type byteStreamTable struct {
	query   uintptr
	addRef  uintptr
	release uintptr
	read    uintptr
	write   uintptr
	seek    uintptr
	setSize uintptr
	copyTo  uintptr
	commit  uintptr
	revert  uintptr
	lock    uintptr
	unlock  uintptr
	stat    uintptr
	clone   uintptr
}

type byteStream struct {
	table *byteStreamTable
	ref   int32
	body  io.ReadCloser
}

var byteStreamMethods = &byteStreamTable{
	query: streamQueryCallback, addRef: streamAddRefCallback, release: streamReleaseCallback,
	read: streamReadCallback, write: streamNotImplemented, seek: streamNotImplemented,
	setSize: streamNotImplemented, copyTo: streamNotImplemented, commit: streamNotImplemented,
	revert: streamNotImplemented, lock: streamNotImplemented, unlock: streamNotImplemented,
	stat: streamNotImplemented, clone: streamNotImplemented,
}

func newByteStream(body io.ReadCloser) uintptr {
	stream := &byteStream{table: byteStreamMethods, ref: 1, body: body}
	pointer := uintptr(unsafe.Pointer(stream))
	liveStreams.Store(pointer, stream)
	return pointer
}

func streamQuery(this, iidPointer, out uintptr) uintptr {
	requested := *(*guid)(unsafe.Pointer(iidPointer))
	if requested == unknownIID || requested == streamIID {
		*(*uintptr)(unsafe.Pointer(out)) = this
		atomic.AddInt32(&(*byteStream)(unsafe.Pointer(this)).ref, 1)
		return 0
	}
	*(*uintptr)(unsafe.Pointer(out)) = 0
	return 0x80004002
}

func streamAddRef(this uintptr) uintptr {
	return uintptr(atomic.AddInt32(&(*byteStream)(unsafe.Pointer(this)).ref, 1))
}

func streamRelease(this uintptr) uintptr {
	stream := (*byteStream)(unsafe.Pointer(this))
	count := atomic.AddInt32(&stream.ref, -1)
	if count == 0 {
		liveStreams.Delete(this)
		stream.body.Close()
	}
	return uintptr(count)
}

func streamReadBytes(this, buffer, count, readOut uintptr) uintptr {
	stream := (*byteStream)(unsafe.Pointer(this))
	if count == 0 {
		return 0
	}
	n, err := stream.body.Read(unsafe.Slice((*byte)(unsafe.Pointer(buffer)), count))
	if readOut != 0 {
		*(*uint32)(unsafe.Pointer(readOut)) = uint32(n)
	}
	if err == io.EOF {
		return 1
	}
	if err != nil {
		return 0x80004005
	}
	return 0
}
