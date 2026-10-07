// Package app runs the windows of one process. A window is a web handler or
// a GUI model, and it may open another window. Run returns when the last
// window closes. Loopback is opt-in: App.NoUI, LEWKIT_NO_UI, or
// ELETROCROMO_NO_UI serves a web handler on a port instead of opening that
// first window. A missing web view is returned. Android keeps that window:
// a GUI model opens the surface and a web handler is the web view. iOS keeps
// a GUI model on the UIKit surface. A web handler on iOS still publishes a
// loopback URL when the host opted into loopback.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/lewkit/x/driver/bundle"
	_ "github.com/lewtec/lewkit/x/driver/bundle/prelude"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
	_ "github.com/lewtec/lewkit/x/driver/webview/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// ErrNilContext means the caller did not pass a context.
var ErrNilContext = errors.New("app: nil context")

// ReadyLinePrefix is printed once the loopback server is listening.
// Packaged hosts parse the URL that follows it.
const ReadyLinePrefix = "ELETROCROMO_READY "

// App opens the first window for the id stamped in x/release.
// Handler is a [Web] view or a [GUI] model.
type App struct {
	Handler Window
	NoUI    bool
	Title   string
	Width   int
	Height  int
}

// Run opens the app window and returns when the last window of the process
// closes. Closing one window leaves the others up. Canceling ctx closes them
// all. A second Run while one is active returns an error.
// A web handler opens a web view. A GUI model opens a host surface. Metal
// presents it on Apple. Vulkan presents it where libvulkan is the screen.
// Loopback runs only when the caller opted in.
func (a App) Run(ctx context.Context) error {
	if ctx == nil {
		return ErrNilContext
	}
	id, _ := release.RequireStamp()
	run := func(ctx context.Context) error {
		return a.open(ctx, id)
	}
	if taskgroup.FromContext(ctx) != nil {
		return run(ctx)
	}
	return entry.Run(ctx, run)
}

func (a App) open(ctx context.Context, id string) error {
	root, err := bundle.ResolveID(ctx, id)
	if err != nil {
		return err
	}
	title := a.Title
	if title == "" {
		title = id
	}
	width, height := a.Width, a.Height
	if width == 0 {
		width = 800
	}
	if height == 0 {
		height = 600
	}
	win := a.Handler
	if win == nil {
		win = Web(nil)
	}
	if _, ok := win.(guiWindow); ok {
		entry.ShowSurface()
	}
	if a.NoUI || envOn("LEWKIT_NO_UI") || envOn("ELETROCROMO_NO_UI") {
		next, err := loopbackWindow(win, nativeHost(win))
		if err != nil {
			return err
		}
		win = next
	}
	return runSession(ctx, win, openCall{
		title:   title,
		width:   width,
		height:  height,
		profile: root.Profile,
	})
}

// loopbackWindow serves a web handler on a desktop packaged host.
// guiNative leaves the window as it is. Android does that for every window.
// iOS does it for a GUI model, which draws on the UIKit surface. A web
// handler on iOS still publishes a loopback URL for the host web view.
func loopbackWindow(win Window, guiNative bool) (Window, error) {
	if guiNative {
		return win, nil
	}
	web, ok := win.(webWindow)
	if !ok {
		return nil, fmt.Errorf("loopback host needs a web handler")
	}
	web.hosted = true
	return web, nil
}

// nativeHost reports whether a packaged no-UI process should keep win.
// Android keeps every window. iOS keeps a GUI model and loopbacks the web handler.
func nativeHost(win Window) bool {
	switch runtime.GOOS {
	case "android":
		return true
	case "ios":
		_, ok := win.(guiWindow)
		return ok
	default:
		return false
	}
}

func serveWeb(ctx context.Context, handler http.Handler) error {
	token := uuid.NewString()
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, BaseContext: func(net.Listener) context.Context { return ctx }}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	go server.Serve(ln)
	link := loopbackLink("http://"+ln.Addr().String(), token)
	announceReady(link)
	<-ctx.Done()
	return nil
}

func announceReady(link string) {
	line := ReadyLinePrefix + link
	fmt.Fprintln(os.Stdout, line)
	_ = os.Stdout.Sync()
	log.Print(line)
	entry.NotifyReady(link)
	path := strings.TrimSpace(os.Getenv("ELETROCROMO_READY_FILE"))
	if path == "" {
		return
	}
	if err := os.WriteFile(path, []byte(link+"\n"), 0o600); err != nil {
		log.Printf("ELETROCROMO_READY_FILE: %v", err)
	}
}

func loopbackLink(raw, token string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := parsed.Hostname()
	if host == "" || host == "localhost" || host == "::1" {
		parsed.Host = net.JoinHostPort("127.0.0.1", parsed.Port())
	}
	base := strings.TrimRight(parsed.String(), "/")
	return base + "/?token=" + token
}

func envOn(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}
