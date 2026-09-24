//go:build darwin

package wkwebview

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/ffi/native/webkit"
	"github.com/lewtec/lewkit/x/thread"
)

const (
	bridgeJavaScript = `window.lewkit={postMessage:function(value){window.webkit.messageHandlers.lewkit.postMessage(value);}};`
	messageName      = "lewkit"
	schemeName       = "app"
	viewHostPrefix   = "view"
	windowStyle      = 1 | 2 | 4 | 8
	backingBuffered  = 2
)

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

var (
	classOnce      sync.Once
	classErr       error
	handlerClass   objc.Class
	pumpOnce       sync.Once
	nextIdentifier atomic.Uint64
	handlers       sync.Map

	selAlloc          = objc.RegisterName("alloc")
	selInit           = objc.RegisterName("init")
	selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	selUTF8String     = objc.RegisterName("UTF8String")
	selDescription    = objc.RegisterName("description")
	selShared         = objc.RegisterName("sharedApplication")
	selSetPolicy      = objc.RegisterName("setActivationPolicy:")
	selSetTitle       = objc.RegisterName("setTitle:")
	selSetContentView = objc.RegisterName("setContentView:")
	selSetDelegate    = objc.RegisterName("setDelegate:")
	selMakeKey        = objc.RegisterName("makeKeyAndOrderFront:")
	selFinishLaunch   = objc.RegisterName("finishLaunching")
	selActivate       = objc.RegisterName("activateIgnoringOtherApps:")
	selCenter         = objc.RegisterName("center")
	selUpdateWindows  = objc.RegisterName("updateWindows")
	selClose          = objc.RegisterName("close")
	selUserContent    = objc.RegisterName("userContentController")
	selAddHandler     = objc.RegisterName("addScriptMessageHandler:name:")
	selAddScript      = objc.RegisterName("addUserScript:")
	selSetScheme      = objc.RegisterName("setURLSchemeHandler:forURLScheme:")
	selInitWebView    = objc.RegisterName("initWithFrame:configuration:")
	selLoadHTML       = objc.RegisterName("loadHTMLString:baseURL:")
	selLoadRequest    = objc.RegisterName("loadRequest:")
	selEvaluate       = objc.RegisterName("evaluateJavaScript:completionHandler:")
	selURLWithString  = objc.RegisterName("URLWithString:")
	selRequestWithURL = objc.RegisterName("requestWithURL:")
	selRequest        = objc.RegisterName("request")
	selURL            = objc.RegisterName("URL")
	selAbsolute       = objc.RegisterName("absoluteString")
	selInitScript     = objc.RegisterName("initWithSource:injectionTime:forMainFrameOnly:")
	selInitResponse   = objc.RegisterName("initWithURL:MIMEType:expectedContentLength:textEncodingName:")
	selDataWithBytes  = objc.RegisterName("dataWithBytes:length:")
	selDidReceive     = objc.RegisterName("didReceiveResponse:")
	selDidReceiveData = objc.RegisterName("didReceiveData:")
	selDidFinish      = objc.RegisterName("didFinish")
	selBody           = objc.RegisterName("body")
	selNextEvent      = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent      = objc.RegisterName("sendEvent:")
	selDistantPast    = objc.RegisterName("distantPast")
)

func frameworks() error {
	return webkit.Load()
}

func (webKitDriver) Open(ctx context.Context, cfg webview.Config) (webview.View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if !thread.Bound() {
		return nil, fmt.Errorf("%w: call thread.Bind from main, then thread.Loop", driver.ErrUnavailable)
	}
	if err := frameworks(); err != nil {
		return nil, err
	}
	if err := registerHandlerClass(); err != nil {
		return nil, err
	}
	view := &webKitView{
		identifier: uintptr(nextIdentifier.Add(1)),
		html:       cfg.HTML,
		files:      cfg.FS,
		handler:    cfg.Handler,
		messages:   make(chan []byte, 32),
		done:       make(chan struct{}),
	}
	var openErr error
	thread.Do(func() {
		openErr = view.create(cfg)
	})
	if openErr != nil {
		return nil, openErr
	}
	pumpOnce.Do(func() {
		thread.OnIdle(pumpEvents)
	})
	context.AfterFunc(ctx, func() { _ = view.Close() })
	return view, nil
}

func registerHandlerClass() error {
	classOnce.Do(func() {
		thread.Do(func() {
			handlerClass, classErr = objc.RegisterClass(
				"LewkitWebKitHandler",
				objc.GetClass("NSObject"),
				nil,
				nil,
				[]objc.MethodDef{
					{Cmd: selInit, Fn: func(self objc.ID, cmd objc.SEL) objc.ID {
						return self.SendSuper(cmd)
					}},
					{Cmd: objc.RegisterName("userContentController:didReceiveScriptMessage:"), Fn: didReceiveScriptMessage},
					{Cmd: objc.RegisterName("webView:startURLSchemeTask:"), Fn: startURLSchemeTask},
					{Cmd: objc.RegisterName("webView:stopURLSchemeTask:"), Fn: stopURLSchemeTask},
					{Cmd: objc.RegisterName("windowWillClose:"), Fn: windowWillClose},
				},
			)
		})
	})
	return classErr
}

func pumpEvents() {
	if !thread.On() {
		return
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(objc.RegisterName("new"))
	defer pool.Send(objc.RegisterName("drain"))
	application := objc.ID(objc.GetClass("NSApplication")).Send(selShared)
	date := objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
	mode := nsString("kCFRunLoopDefaultMode")
	for {
		event := application.Send(selNextEvent, ^uintptr(0), date, mode, true)
		if event == 0 {
			break
		}
		application.Send(selSendEvent, event)
	}
	application.Send(selUpdateWindows)
}

type webKitView struct {
	identifier    uintptr
	html          string
	files         fs.FS
	handler       http.Handler
	messages      chan []byte
	done          chan struct{}
	once          sync.Once
	window        objc.ID
	webView       objc.ID
	nativeHandler objc.ID
}

func (view *webKitView) create(cfg webview.Config) error {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(objc.RegisterName("new"))
	defer pool.Send(objc.RegisterName("drain"))
	width, height, err := cfg.Size()
	if err != nil {
		return err
	}
	application := objc.ID(objc.GetClass("NSApplication")).Send(selShared)
	application.Send(selSetPolicy, 0)
	application.Send(selFinishLaunch)
	handler := objc.ID(handlerClass).Send(selAlloc).Send(selInit)
	if handler == 0 {
		return fmt.Errorf("%w: handler", driver.ErrUnavailable)
	}
	view.nativeHandler = handler
	handlers.Store(uintptr(handler), view)
	configuration := objc.ID(objc.GetClass("WKWebViewConfiguration")).Send(selAlloc).Send(selInit)
	controller := configuration.Send(selUserContent)
	controller.Send(selAddHandler, handler, nsString(messageName))
	script := objc.ID(objc.GetClass("WKUserScript")).Send(selAlloc).Send(selInitScript, nsString(bridgeJavaScript), 0, true)
	controller.Send(selAddScript, script)
	configuration.Send(selSetScheme, handler, nsString(schemeName))
	if err := applyProfile(configuration, cfg.Profile); err != nil {
		return err
	}
	rect := nsRect{Size: nsSize{Width: float64(width), Height: float64(height)}}
	webView := objc.ID(objc.GetClass("WKWebView")).Send(selAlloc).Send(selInitWebView, rect, configuration)
	if webView == 0 {
		return fmt.Errorf("%w: WKWebView", driver.ErrUnavailable)
	}
	window := objc.ID(objc.GetClass("NSWindow")).Send(selAlloc).Send(
		objc.RegisterName("initWithContentRect:styleMask:backing:defer:"),
		rect, windowStyle, backingBuffered, false,
	)
	if window == 0 {
		return fmt.Errorf("%w: window", driver.ErrUnavailable)
	}
	if cfg.Title != "" {
		window.Send(selSetTitle, nsString(cfg.Title))
	}
	window.Send(selSetContentView, webView)
	window.Send(selSetDelegate, handler)
	window.Send(selCenter)
	window.Send(selMakeKey, objc.ID(0))
	window.Send(objc.RegisterName("orderFrontRegardless"))
	application.Send(selActivate, true)
	view.window = window
	view.webView = webView
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

func (view *webKitView) Messages() <-chan []byte { return view.messages }
func (view *webKitView) Done() <-chan struct{}   { return view.done }

func (view *webKitView) Close() error {
	thread.Do(func() {
		if view.window != 0 {
			view.window.Send(selClose)
			return
		}
		view.markClosed()
	})
	return nil
}

func (view *webKitView) markClosed() {
	view.once.Do(func() {
		handlers.Delete(uintptr(view.nativeHandler))
		view.window = 0
		view.webView = 0
		close(view.done)
	})
}

func (view *webKitView) Evaluate(ctx context.Context, script string) (string, error) {
	select {
	case <-view.done:
		return "", webview.ErrClosed
	default:
	}
	call := &evaluation{done: make(chan struct{})}
	thread.Do(func() {
		if view.webView == 0 {
			call.err = webview.ErrClosed
			close(call.done)
			return
		}
		block := objc.NewBlock(func(_ objc.Block, result objc.ID, failure objc.ID) {
			if failure != 0 {
				call.err = fmt.Errorf("%s", cocoaString(failure.Send(selDescription)))
			} else if result != 0 {
				call.text = cocoaString(result.Send(selDescription))
			}
			close(call.done)
		})
		view.webView.Send(selEvaluate, nsString(script), block)
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

func didReceiveScriptMessage(self objc.ID, _ objc.SEL, _ objc.ID, message objc.ID) {
	loaded, ok := handlers.Load(uintptr(self))
	if !ok {
		return
	}
	view := loaded.(*webKitView)
	body := message.Send(selBody)
	text := jsonText(body)
	payload := []byte(text)
	select {
	case view.messages <- payload:
	default:
		go func() {
			select {
			case view.messages <- payload:
			case <-view.done:
			}
		}()
	}
}

func startURLSchemeTask(self objc.ID, _ objc.SEL, _ objc.ID, task objc.ID) {
	loaded, ok := handlers.Load(uintptr(self))
	if !ok {
		return
	}
	view := loaded.(*webKitView)
	request := task.Send(selRequest)
	target := request.Send(selURL)
	parsed, err := url.Parse(cocoaString(target.Send(selAbsolute)))
	if err != nil {
		return
	}
	if view.handler != nil {
		method := cocoaString(request.Send(objc.RegisterName("HTTPMethod")))
		status, header, payload, err := webview.Dispatch(view.handler, method, parsed.String(), nil, httpBody(request))
		if err != nil {
			return
		}
		fields := objc.ID(objc.GetClass("NSMutableDictionary")).Send(objc.RegisterName("dictionary"))
		for name, values := range header {
			for _, value := range values {
				fields.Send(objc.RegisterName("setObject:forKey:"), nsString(value), nsString(name))
			}
		}
		httpResponse := objc.ID(objc.GetClass("NSHTTPURLResponse")).Send(selAlloc).Send(
			objc.RegisterName("initWithURL:statusCode:HTTPVersion:headerFields:"),
			target, status, nsString("HTTP/1.1"), fields,
		)
		task.Send(selDidReceive, httpResponse)
		go deliverBody(task, payload)
		return
	}
	body, contentType, err := webview.ReadPage(view.html, view.files, parsed.Path)
	if err != nil {
		return
	}
	response := objc.ID(objc.GetClass("NSURLResponse")).Send(selAlloc).Send(
		selInitResponse, target, nsString(contentType), len(body), nsString("utf-8"),
	)
	task.Send(selDidReceive, response)
	if len(body) > 0 {
		data := objc.ID(objc.GetClass("NSData")).Send(selDataWithBytes, unsafe.Pointer(&body[0]), len(body))
		task.Send(selDidReceiveData, data)
	}
	task.Send(selDidFinish)
}

func stopURLSchemeTask(objc.ID, objc.SEL, objc.ID, objc.ID) {}

func windowWillClose(self objc.ID, _ objc.SEL, _ objc.ID) {
	if loaded, ok := handlers.Load(uintptr(self)); ok {
		loaded.(*webKitView).markClosed()
	}
}

func applyProfile(configuration objc.ID, profile string) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		return err
	}
	dataConfiguration := newDataStoreConfiguration()
	if dataConfiguration == 0 {
		return fmt.Errorf("webview profile: WebKit data store configuration is unavailable")
	}
	directory := objc.ID(objc.GetClass("NSURL")).Send(objc.RegisterName("fileURLWithPath:isDirectory:"), nsString(profile), true)
	if !sendIfResponds(dataConfiguration, "setGeneralStorageDirectory:", directory) &&
		!sendIfResponds(dataConfiguration, "_setWebStorageDirectory:", directory) {
		return fmt.Errorf("webview profile: this WebKit cannot place storage in %s", profile)
	}
	sendIfResponds(dataConfiguration, "setShouldUseCustomStoragePaths:", 1)
	store := newDataStore(dataConfiguration)
	if store == 0 {
		return fmt.Errorf("webview profile: WebKit did not create a data store for %s", profile)
	}
	configuration.Send(objc.RegisterName("setWebsiteDataStore:"), store)
	return nil
}

func newDataStoreConfiguration() objc.ID {
	for _, name := range []string{"_WKWebsiteDataStoreConfiguration", "WKWebsiteDataStoreConfiguration"} {
		class := objc.GetClass(name)
		if class == 0 {
			continue
		}
		configuration := objc.ID(class).Send(selAlloc).Send(selInit)
		if configuration != 0 {
			return configuration
		}
	}
	return 0
}

func newDataStore(dataConfiguration objc.ID) objc.ID {
	class := objc.GetClass("WKWebsiteDataStore")
	if class == 0 {
		return 0
	}
	selector := objc.RegisterName("_initWithConfiguration:")
	allocated := objc.ID(class).Send(selAlloc)
	if allocated.Send(objc.RegisterName("respondsToSelector:"), selector) != 0 {
		return allocated.Send(selector, dataConfiguration)
	}
	allocated.Send(objc.RegisterName("release"))
	return objc.ID(class).Send(objc.RegisterName("dataStoreWithConfiguration:"), dataConfiguration)
}

func sendIfResponds(object objc.ID, selector string, value any) bool {
	if object == 0 {
		return false
	}
	name := objc.RegisterName(selector)
	if object.Send(objc.RegisterName("respondsToSelector:"), name) == 0 {
		return false
	}
	object.Send(name, value)
	return true
}

func deliverBody(task objc.ID, body io.ReadCloser) {
	defer body.Close()
	buffer := make([]byte, 32*1024)
	for {
		count, err := body.Read(buffer)
		if count > 0 {
			chunk := append([]byte(nil), buffer[:count]...)
			thread.Do(func() {
				data := objc.ID(objc.GetClass("NSData")).Send(selDataWithBytes, unsafe.Pointer(&chunk[0]), len(chunk))
				task.Send(selDidReceiveData, data)
			})
		}
		if err != nil {
			break
		}
	}
	thread.Do(func() { task.Send(selDidFinish) })
}

func httpBody(request objc.ID) io.ReadCloser {
	data := request.Send(objc.RegisterName("HTTPBody"))
	if data == 0 {
		return http.NoBody
	}
	data.Send(objc.RegisterName("retain"))
	length := int(data.Send(objc.RegisterName("length")))
	if length <= 0 {
		data.Send(objc.RegisterName("release"))
		return http.NoBody
	}
	pointer := uintptr(data.Send(objc.RegisterName("bytes")))
	if pointer == 0 {
		data.Send(objc.RegisterName("release"))
		return http.NoBody
	}
	return &dataReader{data: data, payload: unsafe.Slice((*byte)(unsafe.Pointer(pointer)), length)}
}

type dataReader struct {
	data    objc.ID
	payload []byte
}

func (reader *dataReader) Read(payload []byte) (int, error) {
	if len(reader.payload) == 0 {
		return 0, io.EOF
	}
	count := copy(payload, reader.payload)
	reader.payload = reader.payload[count:]
	return count, nil
}

func (reader *dataReader) Close() error {
	if reader.data != 0 {
		reader.data.Send(objc.RegisterName("release"))
		reader.data = 0
	}
	return nil
}

func jsonText(body objc.ID) string {
	if body == 0 {
		return "null"
	}
	if pointer := uintptr(body.Send(selUTF8String)); pointer != 0 {
		return strconv.Quote(goString(pointer))
	}
	return cocoaString(body.Send(selDescription))
}

func nsString(text string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8, text)
}

func cocoaString(value objc.ID) string {
	if value == 0 {
		return ""
	}
	if pointer := uintptr(value.Send(selUTF8String)); pointer != 0 {
		return goString(pointer)
	}
	return ""
}

func goString(pointer uintptr) string {
	if pointer == 0 {
		return ""
	}
	var length int
	for *(*byte)(unsafe.Add(unsafe.Pointer(pointer), length)) != 0 {
		length++
		if length > 1<<20 {
			break
		}
	}
	if length == 0 {
		return ""
	}
	copied := make([]byte, length)
	copy(copied, unsafe.Slice((*byte)(unsafe.Pointer(pointer)), length))
	return string(copied)
}
