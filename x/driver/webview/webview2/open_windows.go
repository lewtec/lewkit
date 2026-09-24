//go:build windows

package webview2

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/colorscheme"
	"github.com/lewtec/lewkit/x/driver/webview"
	native "github.com/lewtec/lewkit/x/ffi/native/webview2"
)

const (
	bridgeJavaScript = `window.lewkit={postMessage:function(value){window.chrome.webview.postMessage(value);}};`
	viewHostSuffix   = ".lewkit.invalid"
	wmClose          = 0x0010
	wmDestroy        = 0x0002
	wmSize           = 0x0005
	wmJob            = 0x8000 + 1
	wsOverlapped     = 0x00CF0000
	wsVisible        = 0x10000000
	swShow           = 5
	cwUseDefault     = 0x80000000
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procCreateWindowEx  = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procGetMessage      = user32.NewProc("GetMessageW")
	procTranslate       = user32.NewProc("TranslateMessage")
	procDispatch        = user32.NewProc("DispatchMessageW")
	procShowWindow      = user32.NewProc("ShowWindow")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procPostMessage     = user32.NewProc("PostMessageW")
	procGetClientRect   = user32.NewProc("GetClientRect")
	procGetModule       = kernel32.NewProc("GetModuleHandleW")

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

	environmentIID        = guid{0xB96D755E, 0x0319, 0x4E92, [8]byte{0xA2, 0x96, 0x23, 0x43, 0x6F, 0x46, 0xA1, 0xFC}}
	controllerIID         = guid{0x4D00C0D1, 0x9434, 0x4EB6, [8]byte{0x80, 0x78, 0x86, 0x97, 0xA5, 0x60, 0x33, 0x4F}}
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

func loader() error { return native.Available() }

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
		identifier: uintptr(nextIdentifier.Add(1)),
		html:       cfg.HTML,
		files:      cfg.FS,
		handler:    cfg.Handler,
		title:      cfg.Title,
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
	post(func() { out <- view.create(ctx) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-out:
		if err != nil {
			return nil, err
		}
	}
	context.AfterFunc(ctx, func() { _ = view.Close() })
	webview.Follow(ctx, func(scheme colorscheme.Scheme) {
		post(func() { view.useScheme(scheme) })
	})
	return view, nil
}

func ensureLoop() error {
	loopOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			if err := native.CoInitialize(); err != nil {
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
			hwnd, _, _ := procCreateWindowEx.Call(0, classAtom, 0, 0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
			if hwnd == 0 {
				loopErr = fmt.Errorf("%w: message window", driver.ErrUnavailable)
				close(loopStarted)
				return
			}
			loopWindow = hwnd
			close(loopStarted)
			var msg message
			for {
				ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
				if int32(ret) <= 0 {
					return
				}
				_, _, _ = procTranslate.Call(uintptr(unsafe.Pointer(&msg)))
				_, _, _ = procDispatch.Call(uintptr(unsafe.Pointer(&msg)))
			}
		}()
	})
	<-loopStarted
	return loopErr
}

func registerWindowClass() error {
	classOnce.Do(func() {
		instance, _, _ := procGetModule.Call(0)
		name, err := syscall.UTF16PtrFromString("lewkit.webview")
		if err != nil {
			classErr = err
			return
		}
		class := wndClassEx{
			wndProc:  windowProcedureCallback,
			instance: instance,
			class:    name,
		}
		class.size = uint32(unsafe.Sizeof(class))
		atom, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))
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
		_, _, _ = procPostMessage.Call(loopWindow, wmJob, 0, 0)
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
	identifier  uintptr
	html        string
	files       fs.FS
	handler     http.Handler
	title       string
	profile     string
	width       int32
	height      int32
	messages    chan []byte
	done        chan struct{}
	once        sync.Once
	hwnd        uintptr
	controller  uintptr
	webView     uintptr
	environment uintptr
}

func (view *edgeView) create(ctx context.Context) error {
	title, err := syscall.UTF16PtrFromString(view.title)
	if err != nil {
		return err
	}
	instance, _, _ := procGetModule.Call(0)
	hwnd, _, _ := procCreateWindowEx.Call(0, classAtom, uintptr(unsafe.Pointer(title)), wsOverlapped|wsVisible, cwUseDefault, cwUseDefault, uintptr(view.width), uintptr(view.height), 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("%w: window", driver.ErrUnavailable)
	}
	view.hwnd = hwnd
	windows.Store(hwnd, view)
	_, _, _ = procShowWindow.Call(hwnd, swShow)
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
	if err := native.CreateEnvironment(folderUTF, uintptr(unsafe.Pointer(environmentHandler))); err != nil {
		return err
	}
	select {
	case view.environment = <-environmentHandler.done:
	case err := <-environmentHandler.failed:
		return err
	}
	controllerHandler := newHandler(controllerHandlerIID, controllerInvokeCallback)
	controllerHandler.done = make(chan uintptr, 1)
	controllerHandler.failed = make(chan error, 1)
	hr := call(view.environment, 3, hwnd, uintptr(unsafe.Pointer(controllerHandler)))
	if hr < 0 {
		return fmt.Errorf("CreateCoreWebView2Controller: %x", uint32(hr))
	}
	select {
	case view.controller = <-controllerHandler.done:
	case err := <-controllerHandler.failed:
		return err
	}
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
	_ = call(view.controller, 4, 1) // put_IsVisible TRUE
	view.resize()
	messageHandler := newHandler(messageHandlerIID, messageInvokeCallback)
	messageHandler.view = view
	var token int64
	hr = call(view.webView, 34, uintptr(unsafe.Pointer(messageHandler)), uintptr(unsafe.Pointer(&token)))
	if hr < 0 {
		return fmt.Errorf("add_WebMessageReceived: %x", uint32(hr))
	}
	resourceHandler := newHandler(resourceHandlerIID, resourceInvokeCallback)
	resourceHandler.view = view
	hr = call(view.webView, 55, uintptr(unsafe.Pointer(resourceHandler)), uintptr(unsafe.Pointer(&token)))
	if hr < 0 {
		return fmt.Errorf("add_WebResourceRequested: %x", uint32(hr))
	}
	filter, err := syscall.UTF16PtrFromString("https://view*" + viewHostSuffix + "/*")
	if err != nil {
		return err
	}
	hr = call(view.webView, 57, uintptr(unsafe.Pointer(filter)), 0)
	if hr < 0 {
		return fmt.Errorf("AddWebResourceRequestedFilter: %x", uint32(hr))
	}
	if scheme, err := colorscheme.Current(ctx); err == nil {
		view.useScheme(scheme)
	}
	target := fmt.Sprintf("https://view%d%s/index.html", view.identifier, viewHostSuffix)
	if view.handler != nil && strings.TrimSpace(view.html) == "" {
		target = fmt.Sprintf("https://view%d%s/", view.identifier, viewHostSuffix)
	}
	targetUTF, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	hr = call(view.webView, 5, uintptr(unsafe.Pointer(targetUTF)))
	if hr < 0 {
		return fmt.Errorf("Navigate: %x", uint32(hr))
	}
	return nil
}

// useScheme sets ICoreWebView2Profile.PreferredColorScheme. That updates
// prefers-color-scheme on the loaded page without a navigation.
// 0 is auto, 1 is light, 2 is dark.
func (view *edgeView) useScheme(scheme colorscheme.Scheme) {
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
	if scheme == colorscheme.Dark {
		value = 2
	}
	_ = call(profile, 9, value)
}

func (view *edgeView) resize() {
	if view.controller == 0 || view.hwnd == 0 {
		return
	}
	var bounds rect
	_, _, _ = procGetClientRect.Call(view.hwnd, uintptr(unsafe.Pointer(&bounds)))
	_ = call(view.controller, 6, uintptr(unsafe.Pointer(&bounds)))
}

func (view *edgeView) Messages() <-chan []byte { return view.messages }
func (view *edgeView) Done() <-chan struct{}   { return view.done }

func (view *edgeView) Close() error {
	post(func() {
		if view.hwnd != 0 {
			_, _, _ = procPostMessage.Call(view.hwnd, wmClose, 0, 0)
			return
		}
		view.markClosed()
	})
	return nil
}

func (view *edgeView) markClosed() {
	view.once.Do(func() {
		windows.Delete(view.hwnd)
		if view.controller != 0 {
			_ = call(view.controller, 24) // Close
			release(view.controller)
			view.controller = 0
		}
		if view.webView != 0 {
			release(view.webView)
			view.webView = 0
		}
		if view.environment != 0 {
			release(view.environment)
			view.environment = 0
		}
		view.hwnd = 0
		close(view.done)
	})
}

func (view *edgeView) Evaluate(ctx context.Context, script string) (string, error) {
	select {
	case <-view.done:
		return "", webview.ErrClosed
	default:
	}
	call := &evaluation{done: make(chan struct{})}
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
	text string
	err  error
	done chan struct{}
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
	if int32(result) < 0 || environment == 0 {
		handler.failed <- fmt.Errorf("environment: %x", uint32(result))
		return 0
	}
	queried, err := query(environment, environmentIID)
	if err != nil {
		handler.failed <- err
		return 0
	}
	handler.done <- queried
	return 0
}

func controllerInvoke(this, result, controller uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	if int32(result) < 0 || controller == 0 {
		handler.failed <- fmt.Errorf("controller: %x", uint32(result))
		return 0
	}
	queried, err := query(controller, controllerIID)
	if err != nil {
		handler.failed <- err
		return 0
	}
	handler.done <- queried
	return 0
}

func messageInvoke(this, _, args uintptr) uintptr {
	handler := (*comHandler)(unsafe.Pointer(this))
	if handler.view == nil || args == 0 {
		return 0
	}
	var textPointer uintptr
	hr := call(args, 4, uintptr(unsafe.Pointer(&textPointer)))
	if hr < 0 || textPointer == 0 {
		return 0
	}
	text := utf16Ptr(textPointer)
	native.FreeTaskMemory(textPointer)
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
	native.FreeTaskMemory(uriPointer)
	parsed, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	if handler.view.handler != nil {
		method := "GET"
		var methodPointer uintptr
		if call(request, 4, uintptr(unsafe.Pointer(&methodPointer))) >= 0 && methodPointer != 0 {
			method = utf16Ptr(methodPointer)
			native.FreeTaskMemory(methodPointer)
		}
		var body io.ReadCloser = http.NoBody
		var content uintptr
		if call(request, 5, uintptr(unsafe.Pointer(&content))) >= 0 && content != 0 {
			body = &comStream{stream: content}
		}
		status, responseHeader, payload, err := webview.Dispatch(handler.view.handler, method, raw, nil, body)
		if err != nil {
			payload.Close()
			return 0
		}
		if strings.Contains(responseHeader.Get("Content-Type"), "html") && (parsed.Path == "/" || parsed.Path == "/index.html") {
			payload = prefixReader(payload, "<script>"+bridgeJavaScript+"</script>")
		}
		writeWebResource(handler.view, args, status, responseHeader, payload)
		return 0
	}
	body, contentType, err := readPage(handler.view.html, handler.view.files, parsed.Path)
	if err != nil {
		return 0
	}
	if parsed.Path == "/" || parsed.Path == "/index.html" {
		body = append([]byte("<script>"+bridgeJavaScript+"</script>"), body...)
	}
	stream, err := native.MemoryStream(body)
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
	if handler.call == nil {
		return 0
	}
	if int32(result) < 0 {
		handler.call.err = fmt.Errorf("script: %x", uint32(result))
	} else if jsonPointer != 0 {
		handler.call.text = utf16Ptr(jsonPointer)
		native.FreeTaskMemory(jsonPointer)
	}
	close(handler.call.done)
	return 0
}

func windowProcedure(hwnd, msg, wparam, lparam uintptr) uintptr {
	if msg == wmJob {
		drainJobs()
		return 0
	}
	loaded, ok := windows.Load(hwnd)
	if !ok {
		ret, _, _ := procDefWindowProc.Call(hwnd, msg, wparam, lparam)
		return ret
	}
	view := loaded.(*edgeView)
	switch msg {
	case wmSize:
		view.resize()
	case wmClose, wmDestroy:
		view.markClosed()
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, msg, wparam, lparam)
	return ret
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

func readPage(html string, files fs.FS, urlPath string) ([]byte, string, error) {
	name, err := webview.AssetName(urlPath)
	if err != nil {
		return nil, "", err
	}
	if name == "index.html" && strings.TrimSpace(html) != "" {
		return []byte(html), "text/html", nil
	}
	if files == nil {
		return nil, "", webview.ErrAsset
	}
	body, err := fs.ReadFile(files, name)
	if err != nil {
		return nil, "", err
	}
	return body, webview.ContentType(name), nil
}
