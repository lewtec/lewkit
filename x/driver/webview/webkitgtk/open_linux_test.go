//go:build linux

package webkitgtk

import (
	"context"
	"net/http"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/stretchr/testify/require"
)

func TestMessageAndAsset(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	files := fstest.MapFS{
		"app.js": &fstest.MapFile{Data: []byte(`window.lewkit.postMessage("asset")`)},
	}
	view, err := gtkDriver{}.Open(ctx, webview.Config{
		Title:   "lewkit webview",
		HTML:    `<!doctype html><script src="app.js"></script>`,
		FS:      files,
		Profile: t.TempDir(),
	})
	if err != nil {
		t.Skip(err)
	}
	defer view.Close()

	got := readMessage(t, view)
	require.Contains(t, got, "asset")
	text, err := view.Evaluate(ctx, "1+2")
	require.NoError(t, err)
	require.Equal(t, "3", text)
}

func TestHandler(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/app.js" {
			response.Header().Set("Content-Type", "text/javascript")
			_, _ = response.Write([]byte(`window.lewkit.postMessage("handler")`))
			return
		}
		response.Header().Set("Content-Type", "text/html")
		_, _ = response.Write([]byte(`<!doctype html><script src="app.js"></script>`))
	})
	view, err := gtkDriver{}.Open(ctx, webview.Config{
		Title:   "lewkit handler",
		Handler: handler,
		Profile: t.TempDir(),
	})
	if err != nil {
		t.Skip(err)
	}
	defer view.Close()
	require.Contains(t, readMessage(t, view), "handler")
}

func readMessage(t *testing.T, view webview.View) string {
	t.Helper()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case b := <-view.Messages():
		return string(b)
	case <-view.Done():
		t.Fatal("window closed")
	case <-timer.C:
		t.Fatal("no script message")
	}
	return ""
}
