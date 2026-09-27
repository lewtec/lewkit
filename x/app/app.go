// Package app opens one window. The window is a web handler or a GUI model.
// A packaged host that sets LEWKIT_NO_UI or ELETROCROMO_NO_UI serves a web
// handler on a loopback port instead.
package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/bundle"
	_ "github.com/lewtec/lewkit/x/driver/bundle/prelude"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
	_ "github.com/lewtec/lewkit/x/driver/webview/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// ReadyLinePrefix is printed once the loopback server is listening.
// Packaged hosts parse the URL that follows it.
const ReadyLinePrefix = "ELETROCROMO_READY "

// App is one window for the id stamped in x/release.
// Handler is a [Web] view or a [GUI] model.
type App struct {
	Handler Window
	NoUI    bool
	Title   string
	Width   int
	Height  int
}

// Run opens the web view, or a loopback server when NoUI or LEWKIT_NO_UI is set.
// The packaged hosts also set ELETROCROMO_NO_UI.
func (a App) Run(ctx context.Context) error {
	driver.SetAppMode(true)
	if ctx == nil {
		ctx = context.Background()
	}
	id, _ := release.RequireStamp()
	run := func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if a.NoUI || envOn("LEWKIT_NO_UI") || envOn("ELETROCROMO_NO_UI") {
			return a.serve(ctx, id)
		}
		return a.desktop(ctx, id)
	}
	if taskgroup.FromContext(ctx) != nil {
		return run(ctx)
	}
	return entry.Run(ctx, run)
}

func (a App) desktop(ctx context.Context, id string) error {
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
	return win.open(ctx, title, width, height, root.Profile)
}

func (a App) serve(ctx context.Context, id string) error {
	_ = id
	token := uuid.NewString()
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	if a.Handler != nil {
		got := a.Handler.httpHandler()
		if got == nil {
			return fmt.Errorf("loopback host needs a web handler")
		}
		handler = got
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
