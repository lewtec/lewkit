//go:build linux && !android

package webkitgtk

import (
	"context"
	"fmt"
	"image"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native/gtk"
	"github.com/lewtec/lewkit/x/ffi/native/webkitgtk"
)

const (
	schemeName     = "app"
	viewHostPrefix = "view"
)

func libraries(ctx context.Context) error {
	if _, err := webkitgtk.Load(ctx); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

type loopState struct {
	symbols *webkitgtk.Symbols
	jobs    chan func()
}

var (
	loopOnce       sync.Once
	loopStateValue *loopState
	loopError      error

	views            sync.Map
	evaluations      sync.Map
	nextIdentifier   atomic.Uint64
	schemeRegistered sync.Once

	closeCallback    = purego.NewCallback(onClose)
	deleteCallback   = purego.NewCallback(onDelete)
	messageCallback  = purego.NewCallback(onMessage)
	schemeCallback   = purego.NewCallback(onScheme)
	evaluateCallback = purego.NewCallback(onEvaluate)
)

func (gtkDriver) Open(ctx context.Context, cfg webview.Config) (webview.View, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: nil context", driver.ErrUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if driver.GetEnv(ctx, "DISPLAY") == "" && driver.GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return nil, fmt.Errorf("%w: no display", driver.ErrUnavailable)
	}
	state, err := ensureLoop(ctx)
	if err != nil {
		return nil, err
	}
	type result struct {
		view webview.View
		err  error
	}
	out := make(chan result, 1)
	state.do(func() {
		view, err := state.open(ctx, cfg)
		out <- result{view, err}
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case opened := <-out:
		if opened.err != nil {
			return nil, opened.err
		}
		context.AfterFunc(ctx, func() { _ = opened.view.Close() })
		webview.Follow(ctx, func(scheme daynight.Mode) {
			state.do(func() { state.symbols.SetPreferDark(scheme == daynight.Dark) })
		})
		return opened.view, nil
	}
}

func ensureLoop(ctx context.Context) (*loopState, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: nil context", driver.ErrUnavailable)
	}
	loopOnce.Do(func() {
		if err := ctx.Err(); err != nil {
			loopError = err
			return
		}
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			loopError = fmt.Errorf("%w: no display", driver.ErrUnavailable)
			return
		}
		symbols, err := webkitgtk.Load(ctx)
		if err != nil {
			loopError = err
			return
		}
		state := &loopState{symbols: symbols}
		if symbols.GTK3() {
			state.jobs = make(chan func(), 32)
		}
		ready := make(chan error, 1)
		boot := func() (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("gtk init: %v", recovered)
				}
			}()
			if !symbols.GTK3() {
				if err = gtk.Ensure(); err != nil {
					return err
				}
			}
			symbols.Init()
			return nil
		}
		switch {
		case symbols.GTK3():
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				if err := boot(); err != nil {
					ready <- err
					return
				}
				ready <- nil
				for {
					state.drain()
					if symbols.Iterate(0) == 0 {
						time.Sleep(5 * time.Millisecond)
					}
				}
			}()
			loopError = <-ready
		case uiOwnsGTK():
			thread.Go(func() { ready <- boot() })
			loopError = <-ready
		case onUIThread():
			loopError = boot()
		default:
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				if err := boot(); err != nil {
					ready <- err
					return
				}
				ready <- nil
				for {
					if gtk.Poll() == 0 {
						time.Sleep(5 * time.Millisecond)
					}
				}
			}()
			loopError = <-ready
		}
		loopStateValue = state
	})
	if loopError != nil {
		return nil, loopError
	}
	return loopStateValue, nil
}

func (state *loopState) do(job func()) {
	if state.jobs != nil {
		state.jobs <- job
		return
	}
	_ = gtk.Enqueue(job)
}

func (state *loopState) drain() {
	if state.jobs == nil {
		return
	}
	for {
		select {
		case job := <-state.jobs:
			job()
		default:
			return
		}
	}
}

func uiOwnsGTK() bool {
	bound, on := uiThread()
	return bound && !on
}

func onUIThread() bool {
	_, on := uiThread()
	return on
}

func uiThread() (bound, on bool) {
	defer func() { _ = recover() }()
	if !thread.Bound() {
		return false, false
	}
	return true, thread.On()
}

func (state *loopState) webView(profile string) (uintptr, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return state.symbols.WebViewNew(), nil
	}
	if err := os.MkdirAll(profile, 0o755); err != nil {
		return 0, err
	}
	cacheDirectory := filepath.Join(profile, "cache")
	if err := os.MkdirAll(cacheDirectory, 0o755); err != nil {
		return 0, err
	}
	return state.symbols.WebViewWithProfile(profile, cacheDirectory)
}

func (state *loopState) open(ctx context.Context, cfg webview.Config) (webview.View, error) {
	width, height, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	identifier := uintptr(nextIdentifier.Add(1))
	window := state.symbols.WindowNew()
	webView, err := state.webView(cfg.Profile)
	if err != nil {
		if window != 0 {
			state.symbols.Destroy(window)
		}
		return nil, err
	}
	if window == 0 || webView == 0 {
		return nil, fmt.Errorf("%w: gtk window", driver.ErrUnavailable)
	}
	view := &gtkView{
		identifier: identifier,
		state:      state,
		webView:    webView,
		window:     window,
		files:      cfg.FS,
		handler:    cfg.Handler,
		messages:   make(chan []byte, 32),
		done:       make(chan struct{}),
	}
	views.Store(identifier, view)
	state.symbols.SetChild(window, webView)
	if cfg.Title != "" {
		state.symbols.SetTitle(window, cfg.Title)
	}
	state.symbols.SetDefaultSize(window, width, height)
	setViewIcon(window, cfg.Icon)
	if state.symbols.GTK3() {
		state.symbols.Connect(window, "delete-event", deleteCallback, identifier)
	} else {
		state.symbols.Connect(window, "close-request", closeCallback, identifier)
	}
	manager := state.symbols.Manager(webView)
	state.symbols.Connect(manager, "script-message-received::"+webview.ScriptName(), messageCallback, identifier)
	if !state.symbols.RegisterMessageHandler(manager, webview.ScriptName()) {
		view.markClosed()
		state.symbols.Destroy(window)
		return nil, fmt.Errorf("%w: message handler", driver.ErrUnavailable)
	}
	state.symbols.AddUserScript(manager, webview.WebKitBridge())
	webContext := state.symbols.Context(webView)
	schemeRegistered.Do(func() {
		state.symbols.RegisterScheme(webContext, schemeName, schemeCallback, 0)
	})
	if scheme, err := daynight.Current(ctx); err == nil {
		state.symbols.SetPreferDark(scheme == daynight.Dark)
	}
	origin := fmt.Sprintf("%s://%s%d/", schemeName, viewHostPrefix, identifier)
	if strings.TrimSpace(cfg.HTML) != "" {
		state.symbols.LoadHTML(webView, cfg.HTML, origin)
	} else if cfg.Handler != nil {
		state.symbols.LoadURI(webView, origin)
	} else {
		state.symbols.LoadURI(webView, origin+"index.html")
	}
	state.symbols.Present(window)
	return view, nil
}

func setViewIcon(widget uintptr, icon image.Image) {
	if icon == nil {
		icon = window.BundleIcon()
	}
	if icon == nil {
		return
	}
	xid, err := gtk.SurfaceXID(widget)
	if err != nil || xid == 0 {
		return
	}
	_ = window.ApplyXIcon(xid, icon)
}

type gtkView struct {
	identifier uintptr
	state      *loopState
	webView    uintptr
	window     uintptr
	files      fs.FS
	handler    http.Handler
	messages   chan []byte
	done       chan struct{}
	once       sync.Once
}

func (view *gtkView) Messages() <-chan []byte { return view.messages }

func (view *gtkView) Done() <-chan struct{} { return view.done }

func (view *gtkView) Close() error {
	view.state.do(func() {
		if view.window != 0 {
			view.state.symbols.Destroy(view.window)
			return
		}
		view.markClosed()
	})
	return nil
}

func (view *gtkView) markClosed() {
	view.once.Do(func() {
		views.Delete(view.identifier)
		view.window = 0
		view.webView = 0
		close(view.done)
	})
}

func (view *gtkView) Evaluate(ctx context.Context, script string) (string, error) {
	select {
	case <-view.done:
		return "", webview.ErrClosed
	default:
	}
	call := &evaluation{done: make(chan struct{})}
	identifier := uintptr(nextIdentifier.Add(1))
	evaluations.Store(identifier, call)
	view.state.do(func() {
		if view.webView == 0 {
			call.err = webview.ErrClosed
			close(call.done)
			evaluations.Delete(identifier)
			return
		}
		view.state.symbols.Evaluate(view.webView, script, evaluateCallback, identifier)
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

func onClose(_, data uintptr) uintptr {
	if loaded, ok := views.Load(data); ok {
		loaded.(*gtkView).markClosed()
	}
	return 0
}

func onDelete(_, _, data uintptr) uintptr {
	onClose(0, data)
	return 0
}

func onMessage(_, value, data uintptr) {
	loaded, ok := views.Load(data)
	if !ok || loopStateValue == nil {
		return
	}
	text := loopStateValue.symbols.ValueText(value)
	view := loaded.(*gtkView)
	body := []byte(text)
	select {
	case view.messages <- body:
	default:
		go func() {
			select {
			case view.messages <- body:
			case <-view.done:
			}
		}()
	}
}

func onScheme(request, _ uintptr) {
	if loopStateValue == nil {
		return
	}
	symbols := loopStateValue.symbols
	parsed, err := url.Parse(symbols.RequestURI(request))
	if err != nil {
		symbols.FinishNotFound(request)
		return
	}
	raw := strings.TrimPrefix(parsed.Host, viewHostPrefix)
	identifier, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		symbols.FinishNotFound(request)
		return
	}
	loaded, ok := views.Load(uintptr(identifier))
	if !ok {
		symbols.FinishNotFound(request)
		return
	}
	view := loaded.(*gtkView)
	if view.handler != nil {
		status, header, body, err := webview.Dispatch(view.handler, symbols.RequestMethod(request), parsed.String(), nil, symbols.RequestBody(request))
		if err != nil {
			symbols.FinishNotFound(request)
			return
		}
		symbols.FinishResponse(request, status, header.Get("Content-Type"), headerPairs(header), body)
		return
	}
	if view.files == nil {
		symbols.FinishNotFound(request)
		return
	}
	name, err := webview.AssetName(parsed.Path)
	if err != nil {
		symbols.FinishNotFound(request)
		return
	}
	body, err := fs.ReadFile(view.files, name)
	if err != nil {
		symbols.FinishNotFound(request)
		return
	}
	symbols.FinishRequest(request, body, webview.ContentType(name))
}

func headerPairs(header http.Header) []webkitgtk.Header {
	if header == nil {
		return nil
	}
	pairs := make([]webkitgtk.Header, 0, len(header))
	for name, values := range header {
		for _, value := range values {
			pairs = append(pairs, webkitgtk.Header{Name: name, Value: value})
		}
	}
	return pairs
}

func onEvaluate(source, result, data uintptr) {
	loaded, ok := evaluations.LoadAndDelete(data)
	if !ok || loopStateValue == nil {
		return
	}
	call := loaded.(*evaluation)
	call.text, call.err = loopStateValue.symbols.FinishEvaluate(source, result)
	close(call.done)
}
