package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Window is one surface the app opens. [Web] is a web view. [GUI] is a
// Vulkan surface driven by a [gui.Model].
type Window interface {
	httpHandler() http.Handler
	open(ctx context.Context, title string, width, height int, profile string) error
}

// Web is a window whose document is handler.
func Web(handler http.Handler) Window { return webWindow{handler: handler} }

// GUI is a window whose picture is model, presented on a Vulkan surface
// when the host can lend one.
func GUI(model gui.Model) Window { return guiWindow{model} }

// Open opens another window and blocks until that window closes.
// During Run, the window joins that session and uses its profile.
// With no session, Open opens one window on its own.
func Open(ctx context.Context, w Window, title string, width, height int) error {
	if w == nil {
		return webview.ErrPage
	}
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	if s := currentSession(); s != nil {
		return s.open(ctx, w, title, width, height)
	}
	return w.open(ctx, title, width, height, "")
}

type webWindow struct {
	handler http.Handler
	hosted  bool
}

func (w webWindow) httpHandler() http.Handler { return w.handler }

func (w webWindow) open(ctx context.Context, title string, width, height int, profile string) error {
	if w.hosted {
		return serveWeb(ctx, w.handler)
	}
	view, err := webview.Open(ctx, webview.Config{
		Title:   title,
		Width:   width,
		Height:  height,
		Profile: profile,
		Handler: w.handler,
	})
	if err != nil {
		if errors.Is(err, driver.ErrUnavailable) {
			return serveWeb(ctx, w.handler)
		}
		return err
	}
	defer view.Close()
	select {
	case <-ctx.Done():
		return nil
	case <-view.Done():
		return nil
	}
}

type guiWindow struct{ model gui.Model }

func (guiWindow) httpHandler() http.Handler { return nil }

func (w guiWindow) open(ctx context.Context, title string, width, height int, _ string) error {
	return gui.Open(ctx, w.model, gui.Options{
		Title:  title,
		Width:  width,
		Height: height,
	})
}
