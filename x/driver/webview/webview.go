package webview

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

const (
	defaultWidth  = 640
	defaultHeight = 480
)

var (
	// ErrPage means Config has neither HTML nor FS.
	ErrPage = errors.New("webview page is empty")
	// ErrSize means a dimension is negative.
	ErrSize = errors.New("invalid size")
	// ErrAsset means a request path cannot be opened from the page FS.
	ErrAsset = errors.New("webview asset")
	// ErrClosed means the view is gone.
	ErrClosed = errors.New("webview closed")
)

// Config is the window and the document it shows.
// HTML is loaded from memory. FS, when set, answers app:// requests
// for that view. Relative URLs in HTML resolve against that origin.
// Profile is the directory for cookies, storage, and cache. Empty uses
// the web view's own default directory.
// Handler, when set, answers each request on the view origin in-process.
type Config struct {
	Title   string
	Width   int
	Height  int
	HTML    string
	FS      fs.FS
	Profile string
	Handler http.Handler
}

// Size returns the client size. Zero becomes 640×480.
func (c Config) Size() (int, int, error) {
	if c.Width < 0 || c.Height < 0 {
		return 0, 0, ErrSize
	}
	width, height := c.Width, c.Height
	if width == 0 {
		width = defaultWidth
	}
	if height == 0 {
		height = defaultHeight
	}
	return width, height, nil
}

// Validate reports a config that cannot open.
func (c Config) Validate() error {
	if _, _, err := c.Size(); err != nil {
		return err
	}
	if strings.TrimSpace(c.HTML) == "" && c.FS == nil && c.Handler == nil {
		return ErrPage
	}
	return nil
}

// AssetName maps a URL path onto an fs.FS name. "/" is index.html.
func AssetName(urlPath string) (string, error) {
	if urlPath == "" || urlPath == "/" {
		return "index.html", nil
	}
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" || name == "." {
		return "index.html", nil
	}
	if !fs.ValidPath(name) {
		return "", fmt.Errorf("%w: %s", ErrAsset, urlPath)
	}
	return name, nil
}

// ContentType returns a content type for a file name.
func ContentType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js", ".mjs":
		return "text/javascript"
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".wasm":
		return "application/wasm"
	case ".txt":
		return "text/plain"
	case ".woff2":
		return "font/woff2"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// View is one OS web view.
//
// Messages delivers JSON text from window.lewkit.post / postMessage.
// Evaluate runs JavaScript in the page and returns JSON text.
// Done closes when the window closes.
type View interface {
	Evaluate(ctx context.Context, script string) (string, error)
	Messages() <-chan []byte
	Done() <-chan struct{}
	Close() error
}

// Driver opens a view.
type Driver interface {
	Open(ctx context.Context, cfg Config) (View, error)
}

// Open asks the active webview driver for a view.
func Open(ctx context.Context, cfg Config) (View, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return driver.WithResult(ctx, func(d Driver) (View, error) {
		return d.Open(ctx, cfg)
	})
}
