// Astro SSR (Cloudflare adapter) hosted by orvalho workers.
// Guest script and client assets are embedded. Produce them with: mise run build
//
//	go run ./cmd/lewkit release run --config ./examples/astro/eletrocromo.json
package main

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

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	assets, err := fs.Sub(assetsRoot, "embed/assets")
	if err != nil {
		return err
	}
	iso := workers.New(guestJS, workers.Options{
		Env: map[string]string{"SITE": "eletrocromo-astro"},
		Bindings: map[string]workers.Binding{
			"ASSETS": workers.NewAssetBinding(assets, "."),
		},
		Fetch: workers.HTTPFetch(workers.EgressList{
			"catfact.ninja",
			"https://catfact.ninja",
		}, nil, 0),
	})
	h := workers.Handler(iso)
	return app.App{
		Title: "Astro",
		Handler: app.Web(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
			h.ServeHTTP(rw, r)
			if rw.code >= 500 {
				slog.Error("astro", "method", r.Method, "uri", r.URL.RequestURI(), "status", rw.code)
			}
		})),
	}.Run(ctx)
}
