//go:build linux

package webkitgtk

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means a required shared library or symbol is missing.
var ErrUnavailable = errors.New("webkitgtk unavailable")

// Symbols is the WebKitGTK 6 / GTK 4 entry points used to open a page
// and pass script messages. Callers run these on one OS thread.
type Symbols struct {
	init                func()
	iterate             func(ctx uintptr, block int32) int32
	wake                func(ctx uintptr)
	pending             func(ctx uintptr) int32
	signal              func(obj uintptr, name *byte, handler, data, destroy uintptr, flags uint32) uint64
	unref               func(obj uintptr)
	free                func(p uintptr)
	memoryStream        func(data uintptr, length int64, destroy uintptr) uintptr
	ioErrorQuark        func() uint32
	newError            func(domain uint32, code int32, message *byte) uintptr
	freeError           func(gError uintptr)
	windowNew           func() uintptr
	settingsDefault     func() uintptr
	valueInit           func(value uintptr, gtype uintptr)
	valueSetBool        func(value uintptr, v int32)
	valueUnset          func(value uintptr)
	setProperty         func(object uintptr, name *byte, value uintptr)
	setTitle            func(window uintptr, title *byte)
	setSize             func(window uintptr, width, height int32)
	setChild            func(window, child uintptr)
	present             func(window uintptr)
	destroy             func(window uintptr)
	newWebView          func() uintptr
	webViewType         func() uintptr
	objectNew           func(objectType uintptr, property *byte, value, terminator uintptr) uintptr
	networkSession      func(dataDirectory, cacheDirectory *byte) uintptr
	loadHTML            func(view uintptr, html, base *byte)
	loadURI             func(view uintptr, uri *byte)
	getManager          func(view uintptr) uintptr
	getContext          func(view uintptr) uintptr
	registerMessage     func(manager uintptr, name, world *byte) int32
	newUserScript       func(source uintptr, frames, when int32, allow, block uintptr) uintptr
	unrefUserScript     func(script uintptr)
	addUserScript       func(manager, script uintptr)
	evaluateJavaScript  func(view uintptr, script *byte, length int64, world, source, cancel, callback, data uintptr)
	finishJavaScript    func(view, result, errorOut uintptr) uintptr
	valueToJSON         func(value uintptr, indent uint32) uintptr
	valueToString       func(value uintptr) uintptr
	securityManager     func(webContext uintptr) uintptr
	registerSecure      func(manager uintptr, scheme *byte)
	registerCORS        func(manager uintptr, scheme *byte)
	registerLocal       func(manager uintptr, scheme *byte)
	registerScheme      func(webContext uintptr, scheme *byte, callback, data, destroy uintptr)
	requestURI          func(request uintptr) uintptr
	requestMethod       func(request uintptr) uintptr
	requestBody         func(request uintptr) uintptr
	streamRead          func(stream, buffer uintptr, count, cancellable, errorOut uintptr) int64
	unixInputStream     func(fd, closeFD int32) uintptr
	responseNew         func(stream uintptr, length int64) uintptr
	responseStatus      func(response uintptr, status uint32, reason *byte)
	responseContentType func(response uintptr, contentType *byte)
	responseHeaders     func(response, headers uintptr)
	finishWithResponse  func(request, response uintptr)
	headersNew          func(kind int32) uintptr
	headersAppend       func(headers uintptr, name, value *byte)
	finishRequest       func(request, stream uintptr, length int64, mime *byte)
	finishRequestError  func(request, gError uintptr)
}

// Load opens the system WebKitGTK 6 stack. WEBKITGTK_LIB, when set, is a
// colon-separated list of directories searched before the NixOS system
// profile (/run/current-system/sw/lib) and the loader path.
func Load() (*Symbols, error) {
	glib, err := openOne("libglib-2.0.so.0")
	if err != nil {
		return nil, err
	}
	gobject, err := openOne("libgobject-2.0.so.0")
	if err != nil {
		return nil, err
	}
	gio, err := openOne("libgio-2.0.so.0")
	if err != nil {
		return nil, err
	}
	gtk, err := openOne("libgtk-4.so.1")
	if err != nil {
		return nil, err
	}
	jsc, err := openOne("libjavascriptcoregtk-6.0.so.1")
	if err != nil {
		return nil, err
	}
	web, err := openOne("libwebkitgtk-6.0.so.4")
	if err != nil {
		return nil, err
	}
	s := &Symbols{}
	if err := bind(glib, "g_main_context_iteration", &s.iterate); err != nil {
		return nil, err
	}
	if err := bind(glib, "g_main_context_wakeup", &s.wake); err != nil {
		return nil, err
	}
	if err := bind(glib, "g_main_context_pending", &s.pending); err != nil {
		return nil, err
	}
	if err := bind(glib, "g_free", &s.free); err != nil {
		return nil, err
	}
	if err := bind(glib, "g_error_new_literal", &s.newError); err != nil {
		return nil, err
	}
	if err := bind(glib, "g_error_free", &s.freeError); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_signal_connect_data", &s.signal); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_object_unref", &s.unref); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_value_init", &s.valueInit); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_value_set_boolean", &s.valueSetBool); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_value_unset", &s.valueUnset); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_object_set_property", &s.setProperty); err != nil {
		return nil, err
	}
	if err := bind(gobject, "g_object_new", &s.objectNew); err != nil {
		return nil, err
	}
	if err := bind(gio, "g_memory_input_stream_new_from_data", &s.memoryStream); err != nil {
		return nil, err
	}
	if err := bind(gio, "g_io_error_quark", &s.ioErrorQuark); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_init", &s.init); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_settings_get_default", &s.settingsDefault); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_new", &s.windowNew); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_set_title", &s.setTitle); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_set_default_size", &s.setSize); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_set_child", &s.setChild); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_present", &s.present); err != nil {
		return nil, err
	}
	if err := bind(gtk, "gtk_window_destroy", &s.destroy); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_new", &s.newWebView); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_get_type", &s.webViewType); err != nil {
		return nil, err
	}
	_ = tryBind(web, "webkit_network_session_new", &s.networkSession)
	if err := bind(web, "webkit_web_view_load_html", &s.loadHTML); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_load_uri", &s.loadURI); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_get_user_content_manager", &s.getManager); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_get_context", &s.getContext); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_user_content_manager_register_script_message_handler", &s.registerMessage); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_user_script_new", &s.newUserScript); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_user_script_unref", &s.unrefUserScript); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_user_content_manager_add_script", &s.addUserScript); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_evaluate_javascript", &s.evaluateJavaScript); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_view_evaluate_javascript_finish", &s.finishJavaScript); err != nil {
		return nil, err
	}
	if err := bind(jsc, "jsc_value_to_json", &s.valueToJSON); err != nil {
		return nil, err
	}
	if err := bind(jsc, "jsc_value_to_string", &s.valueToString); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_context_get_security_manager", &s.securityManager); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_security_manager_register_uri_scheme_as_secure", &s.registerSecure); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_security_manager_register_uri_scheme_as_cors_enabled", &s.registerCORS); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_security_manager_register_uri_scheme_as_local", &s.registerLocal); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_web_context_register_uri_scheme", &s.registerScheme); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_uri_scheme_request_get_uri", &s.requestURI); err != nil {
		return nil, err
	}
	_ = tryBind(web, "webkit_uri_scheme_request_get_http_method", &s.requestMethod)
	_ = tryBind(web, "webkit_uri_scheme_request_get_http_body", &s.requestBody)
	_ = tryBind(gio, "g_input_stream_read", &s.streamRead)
	_ = tryBind(gio, "g_unix_input_stream_new", &s.unixInputStream)
	_ = tryBind(web, "webkit_uri_scheme_response_new", &s.responseNew)
	_ = tryBind(web, "webkit_uri_scheme_response_set_status", &s.responseStatus)
	_ = tryBind(web, "webkit_uri_scheme_response_set_content_type", &s.responseContentType)
	_ = tryBind(web, "webkit_uri_scheme_response_set_http_headers", &s.responseHeaders)
	_ = tryBind(web, "webkit_uri_scheme_request_finish_with_response", &s.finishWithResponse)
	if soup, err := openOne("libsoup-3.0.so.0"); err == nil {
		_ = tryBind(soup, "soup_message_headers_new", &s.headersNew)
		_ = tryBind(soup, "soup_message_headers_append", &s.headersAppend)
	}
	if err := bind(web, "webkit_uri_scheme_request_finish", &s.finishRequest); err != nil {
		return nil, err
	}
	if err := bind(web, "webkit_uri_scheme_request_finish_error", &s.finishRequestError); err != nil {
		return nil, err
	}
	return s, nil
}

// nixOSSystemLib is the NixOS system profile library directory.
const nixOSSystemLib = "/run/current-system/sw/lib"

func libraryDirectories() []string {
	var dirs []string
	for _, dir := range strings.Split(os.Getenv("WEBKITGTK_LIB"), ":") {
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return append(dirs, nixOSSystemLib)
}

func openOne(soname string) (uintptr, error) {
	var last error
	for _, dir := range libraryDirectories() {
		lib, err := native.Open(filepath.Join(dir, soname), native.Global|native.Lazy)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	lib, err := native.Open(soname, native.Global|native.Lazy)
	if err == nil {
		return lib, nil
	}
	if last != nil {
		err = last
	}
	return 0, fmt.Errorf("%w: %s: %v", ErrUnavailable, soname, err)
}

func bind(lib uintptr, name string, fnptr any) error {
	if err := tryBind(lib, name, fnptr); err != nil {
		return fmt.Errorf("%w: %s", ErrUnavailable, name)
	}
	return nil
}

func tryBind(lib uintptr, name string, fnptr any) error {
	if _, err := native.Symbol(lib, name); err != nil {
		return err
	}
	native.Func(lib, name, fnptr)
	return nil
}

// Init prepares GTK on the calling thread.
func (s *Symbols) Init() { s.init() }

// Iterate dispatches the default main context. block is 1 to wait.
func (s *Symbols) Iterate(block int32) int32 { return s.iterate(0, block) }

// Wake interrupts a blocking Iterate.
func (s *Symbols) Wake() { s.wake(0) }

// Connect installs handler for the detailed signal. data is passed back
// as the last callback argument. handler is a purego callback.
func (s *Symbols) Connect(obj uintptr, signal string, handler, data uintptr) {
	name := cStringBytes(signal)
	s.signal(obj, cStringPointer(name), handler, data, 0, 0)
	runtime.KeepAlive(name)
}

// WindowNew returns a GtkWindow.
func (s *Symbols) WindowNew() uintptr { return s.windowNew() }

// SetTitle sets the window title.
func (symbols *Symbols) SetTitle(window uintptr, title string) {
	titleBytes := cStringBytes(title)
	symbols.setTitle(window, cStringPointer(titleBytes))
	runtime.KeepAlive(titleBytes)
}

// SetDefaultSize sets the initial client size.
func (symbols *Symbols) SetDefaultSize(window uintptr, width, height int) {
	symbols.setSize(window, int32(width), int32(height))
}

// SetChild parents child in the window.
func (symbols *Symbols) SetChild(window, child uintptr) { symbols.setChild(window, child) }

// Present shows the window.
func (symbols *Symbols) Present(window uintptr) { symbols.present(window) }

// gTypeBoolean is G_TYPE_BOOLEAN on LP64 (5 << 2).
const gTypeBoolean = 20

type gValue struct {
	gType uintptr
	data0 uint64
	data1 uint64
}

// SetPreferDark sets gtk-application-prefer-dark on the default GtkSettings.
// WebKitGTK reads that property for prefers-color-scheme, including on a page
// that is already loaded.
func (symbols *Symbols) SetPreferDark(dark bool) {
	if symbols == nil || symbols.settingsDefault == nil || symbols.setProperty == nil {
		return
	}
	settings := symbols.settingsDefault()
	if settings == 0 {
		return
	}
	var value gValue
	symbols.valueInit(uintptr(unsafe.Pointer(&value)), gTypeBoolean)
	flag := int32(0)
	if dark {
		flag = 1
	}
	symbols.valueSetBool(uintptr(unsafe.Pointer(&value)), flag)
	name := cStringBytes("gtk-application-prefer-dark")
	symbols.setProperty(settings, cStringPointer(name), uintptr(unsafe.Pointer(&value)))
	symbols.valueUnset(uintptr(unsafe.Pointer(&value)))
	runtime.KeepAlive(name)
	runtime.KeepAlive(&value)
}

// Destroy closes the window.
func (symbols *Symbols) Destroy(window uintptr) {
	if window != 0 {
		symbols.destroy(window)
	}
}

// WebViewNew returns a WebKitWebView on the default profile.
func (symbols *Symbols) WebViewNew() uintptr { return symbols.newWebView() }

// WebViewWithProfile returns a WebKitWebView whose cookies and storage
// live in dataDirectory. cacheDirectory holds the network cache.
func (symbols *Symbols) WebViewWithProfile(dataDirectory, cacheDirectory string) (uintptr, error) {
	if symbols.networkSession == nil || symbols.objectNew == nil || symbols.webViewType == nil {
		return 0, fmt.Errorf("%w: network session", ErrUnavailable)
	}
	dataBytes := cStringBytes(dataDirectory)
	cacheBytes := cStringBytes(cacheDirectory)
	session := symbols.networkSession(cStringPointer(dataBytes), cStringPointer(cacheBytes))
	runtime.KeepAlive(dataBytes)
	runtime.KeepAlive(cacheBytes)
	if session == 0 {
		return 0, fmt.Errorf("%w: network session", ErrUnavailable)
	}
	property := cStringBytes("network-session")
	webView := symbols.objectNew(symbols.webViewType(), cStringPointer(property), session, 0)
	runtime.KeepAlive(property)
	if webView == 0 {
		return 0, fmt.Errorf("%w: web view", ErrUnavailable)
	}
	return webView, nil
}

// LoadHTML loads a document in memory. base may be empty.
func (symbols *Symbols) LoadHTML(view uintptr, html, base string) {
	htmlBytes := cStringBytes(html)
	var baseBytes []byte
	var basePointer *byte
	if base != "" {
		baseBytes = cStringBytes(base)
		basePointer = cStringPointer(baseBytes)
	}
	symbols.loadHTML(view, cStringPointer(htmlBytes), basePointer)
	runtime.KeepAlive(htmlBytes)
	runtime.KeepAlive(baseBytes)
}

// LoadURI navigates to uri.
func (symbols *Symbols) LoadURI(view uintptr, uri string) {
	uriBytes := cStringBytes(uri)
	symbols.loadURI(view, cStringPointer(uriBytes))
	runtime.KeepAlive(uriBytes)
}

// Manager returns the view's user content manager.
func (symbols *Symbols) Manager(view uintptr) uintptr { return symbols.getManager(view) }

// Context returns the view's web context.
func (symbols *Symbols) Context(view uintptr) uintptr { return symbols.getContext(view) }

// RegisterMessageHandler enables window.webkit.messageHandlers.<name>.
func (symbols *Symbols) RegisterMessageHandler(manager uintptr, name string) bool {
	nameBytes := cStringBytes(name)
	ok := symbols.registerMessage(manager, cStringPointer(nameBytes), nil)
	runtime.KeepAlive(nameBytes)
	return ok != 0
}

// AddUserScript injects source at document start in the top frame.
func (symbols *Symbols) AddUserScript(manager uintptr, source string) {
	sourceBytes := cStringBytes(source)
	script := symbols.newUserScript(uintptr(unsafe.Pointer(cStringPointer(sourceBytes))), 1, 0, 0, 0)
	runtime.KeepAlive(sourceBytes)
	if script == 0 {
		return
	}
	symbols.addUserScript(manager, script)
	symbols.unrefUserScript(script)
}

// Evaluate runs script. The callback runs on this thread when it finishes.
func (symbols *Symbols) Evaluate(view uintptr, script string, callback, data uintptr) {
	scriptBytes := cStringBytes(script)
	symbols.evaluateJavaScript(view, cStringPointer(scriptBytes), -1, 0, 0, 0, callback, data)
	runtime.KeepAlive(scriptBytes)
}

// FinishEvaluate reads the JavaScript result.
func (symbols *Symbols) FinishEvaluate(view, result uintptr) (string, error) {
	var slot uintptr
	value := symbols.finishJavaScript(view, result, uintptr(unsafe.Pointer(&slot)))
	if slot != 0 {
		message := goString(errorMessage(slot))
		symbols.freeError(slot)
		if value != 0 {
			symbols.unref(value)
		}
		if message == "" {
			message = "javascript failed"
		}
		return "", fmt.Errorf("%s", message)
	}
	if value == 0 {
		return "", nil
	}
	defer symbols.unref(value)
	if pointer := symbols.valueToJSON(value, 0); pointer != 0 {
		text := goString(pointer)
		symbols.free(pointer)
		return text, nil
	}
	if pointer := symbols.valueToString(value); pointer != 0 {
		text := goString(pointer)
		symbols.free(pointer)
		return text, nil
	}
	return "", nil
}

// RegisterScheme installs callback for scheme and marks it local, secure,
// and CORS-enabled so the page can load scripts from it.
func (symbols *Symbols) RegisterScheme(webContext uintptr, scheme string, callback, data uintptr) {
	schemeBytes := cStringBytes(scheme)
	if manager := symbols.securityManager(webContext); manager != 0 {
		symbols.registerSecure(manager, cStringPointer(schemeBytes))
		symbols.registerCORS(manager, cStringPointer(schemeBytes))
		symbols.registerLocal(manager, cStringPointer(schemeBytes))
	}
	symbols.registerScheme(webContext, cStringPointer(schemeBytes), callback, data, 0)
	runtime.KeepAlive(schemeBytes)
}

// ValueText is the JSON form of a JSCValue, or its string form.
// The value is not unreffed.
func (symbols *Symbols) ValueText(value uintptr) string {
	if value == 0 {
		return ""
	}
	if pointer := symbols.valueToJSON(value, 0); pointer != 0 {
		text := goString(pointer)
		symbols.free(pointer)
		return text
	}
	if pointer := symbols.valueToString(value); pointer != 0 {
		text := goString(pointer)
		symbols.free(pointer)
		return text
	}
	return ""
}

// RequestURI returns the URI of a scheme request.
func (symbols *Symbols) RequestURI(request uintptr) string {
	return goString(symbols.requestURI(request))
}

// RequestMethod returns the HTTP method, or GET when the library has none.
func (symbols *Symbols) RequestMethod(request uintptr) string {
	if symbols.requestMethod == nil {
		return "GET"
	}
	method := goString(symbols.requestMethod(request))
	if method == "" {
		return "GET"
	}
	return method
}

// RequestBody is the scheme request body. Close it when the handler returns.
// A missing body is an empty reader.
func (symbols *Symbols) RequestBody(request uintptr) io.ReadCloser {
	if symbols.requestBody == nil || symbols.streamRead == nil {
		return http.NoBody
	}
	stream := symbols.requestBody(request)
	if stream == 0 {
		return http.NoBody
	}
	return &inputStream{symbols: symbols, stream: stream}
}

type inputStream struct {
	symbols *Symbols
	stream  uintptr
}

func (stream *inputStream) Read(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}
	var slot uintptr
	count := stream.symbols.streamRead(stream.stream, uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)), 0, uintptr(unsafe.Pointer(&slot)))
	if slot != 0 {
		message := goString(errorMessage(slot))
		stream.symbols.freeError(slot)
		if message == "" {
			message = "read"
		}
		return 0, fmt.Errorf("%s", message)
	}
	if count <= 0 {
		return 0, io.EOF
	}
	return int(count), nil
}

func (stream *inputStream) Close() error {
	if stream.stream != 0 {
		stream.symbols.unref(stream.stream)
		stream.stream = 0
	}
	return nil
}

// Header is one response header line pair.
type Header struct {
	Name  string
	Value string
}

// FinishResponse completes a scheme request with an HTTP status and headers.
// body is read as WebKit pulls bytes. FinishResponse closes body.
func (symbols *Symbols) FinishResponse(request uintptr, status int, contentType string, headers []Header, body io.Reader) {
	if body == nil {
		body = http.NoBody
	}
	stream, length := symbols.inputStream(body)
	if symbols.responseNew == nil || symbols.finishWithResponse == nil || symbols.responseStatus == nil || stream == 0 {
		symbols.FinishRequest(request, nil, contentType)
		return
	}
	response := symbols.responseNew(stream, length)
	if response == 0 {
		symbols.FinishRequest(request, nil, contentType)
		return
	}
	reason := cStringBytes(httpReason(status))
	symbols.responseStatus(response, uint32(status), cStringPointer(reason))
	runtime.KeepAlive(reason)
	if contentType != "" && symbols.responseContentType != nil {
		mime := cStringBytes(contentType)
		symbols.responseContentType(response, cStringPointer(mime))
		runtime.KeepAlive(mime)
	}
	if symbols.headersNew != nil && symbols.headersAppend != nil && symbols.responseHeaders != nil && len(headers) > 0 {
		soupHeaders := symbols.headersNew(1)
		for _, header := range headers {
			name := cStringBytes(header.Name)
			value := cStringBytes(header.Value)
			symbols.headersAppend(soupHeaders, cStringPointer(name), cStringPointer(value))
			runtime.KeepAlive(name)
			runtime.KeepAlive(value)
		}
		symbols.responseHeaders(response, soupHeaders)
	}
	symbols.finishWithResponse(request, response)
}

func (symbols *Symbols) inputStream(body io.Reader) (uintptr, int64) {
	if symbols.unixInputStream == nil {
		if closer, ok := body.(io.Closer); ok {
			closer.Close()
		}
		return 0, 0
	}
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		if closer, ok := body.(io.Closer); ok {
			closer.Close()
		}
		return 0, 0
	}
	go func() {
		defer writeFile.Close()
		_, _ = io.Copy(writeFile, body)
		if closer, ok := body.(io.Closer); ok {
			closer.Close()
		}
	}()
	descriptor, err := syscall.Dup(int(readFile.Fd()))
	readFile.Close()
	if err != nil {
		writeFile.Close()
		return 0, 0
	}
	return symbols.unixInputStream(int32(descriptor), 1), -1
}

func httpReason(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "OK"
	}
	return text
}

// FinishRequest completes a scheme request with body. WebKit takes stream.
func (symbols *Symbols) FinishRequest(request uintptr, body []byte, mime string) {
	var pointer uintptr
	length := int64(len(body))
	if length > 0 {
		pointer = pinBytes(body)
	}
	stream := symbols.memoryStream(pointer, length, destroyPinnedBytes)
	mimeBytes := cStringBytes(mime)
	symbols.finishRequest(request, stream, length, cStringPointer(mimeBytes))
	runtime.KeepAlive(mimeBytes)
}

// FinishNotFound completes a scheme request as not-found.
func (symbols *Symbols) FinishNotFound(request uintptr) {
	message := cStringBytes("not found")
	gError := symbols.newError(symbols.ioErrorQuark(), 1, cStringPointer(message))
	runtime.KeepAlive(message)
	if gError == 0 {
		symbols.FinishRequest(request, nil, "text/plain")
		return
	}
	symbols.finishRequestError(request, gError)
}

func errorMessage(gError uintptr) uintptr {
	// GError { uint32 domain; int32 code; char *message } on 64-bit.
	type glibError struct {
		Domain  uint32
		Code    int32
		Message uintptr
	}
	return (*glibError)(unsafe.Pointer(gError)).Message
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
