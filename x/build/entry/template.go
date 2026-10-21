package entry

import (
	"fmt"
	"strings"
)

func displayTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "app"
	}
	return title
}

func staticZipMain(title string) string {
	title = displayTitle(title)
	return fmt.Sprintf(`package main

import (
	"bytes"
	"context"
	_ "embed"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/site"
)

//go:embed site.zip
var siteZip []byte

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	handler, err := site.Zip(ctx, bytes.NewReader(siteZip))
	if err != nil {
		return err
	}
	return app.App{Title: %q, Handler: app.Web(handler)}.Run(ctx)
}
`, title)
}

func staticDirMain(title string) string {
	title = displayTitle(title)
	return fmt.Sprintf(`package main

import (
	"context"
	"embed"
	"io/fs"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/site"
)

//go:embed all:site
var siteRoot embed.FS

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	files, err := fs.Sub(siteRoot, "site")
	if err != nil {
		return err
	}
	return app.App{Title: %q, Handler: app.Web(site.Dir(files))}.Run(ctx)
}
`, title)
}

func astroMain(title string) string {
	title = displayTitle(title)
	return fmt.Sprintf(`package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lucasew/orvalho/pkg/workers"
)

//go:embed embed/guest.js
var guestJS string

//go:embed all:embed/assets
var assetsRoot embed.FS

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	assets, err := fs.Sub(assetsRoot, "embed/assets")
	if err != nil {
		return err
	}
	iso := workers.New(guestJS, workers.Options{
		Env: map[string]string{"SITE": %q},
		Bindings: map[string]workers.Binding{
			"ASSETS": workers.NewAssetBinding(assets, "."),
		},
		Fetch: workers.HTTPFetch(workers.EgressList{"*"}, nil, 0),
	})
	handler := workers.Handler(iso)
	return app.App{
		Title: %q,
		Handler: app.Web(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			recorder := &statusRecorder{ResponseWriter: response, code: http.StatusOK}
			handler.ServeHTTP(recorder, request)
			if recorder.code >= 500 {
				slog.Error("astro", "method", request.Method, "uri", request.URL.RequestURI(), "status", recorder.code)
			}
		})),
	}.Run(ctx)
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (recorder *statusRecorder) WriteHeader(code int) {
	recorder.code = code
	recorder.ResponseWriter.WriteHeader(code)
}
`, title, title)
}
