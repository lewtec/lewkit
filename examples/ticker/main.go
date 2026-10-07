// Ticker adds one every second. The page only reads that count.
//
//	go run ./cmd/lewkit release run --config ./examples/ticker/eletrocromo.json
package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

func runApp(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var count atomic.Int64
	go tick(ctx, &count)
	return app.App{
		Title:   "Ticker",
		Handler: app.Web(newTicker(&count)),
	}.Run(ctx)
}

func tick(ctx context.Context, count *atomic.Int64) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			count.Add(1)
		}
	}
}

var page = template.Must(template.New("ticker").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="1">
  <title>Ticker</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
    body { max-width: 28rem; margin: 3rem auto; padding: 0 1rem; text-align: center; }
    h1 { font-size: 1.25rem; font-weight: 600; }
    .count { font-size: 4rem; font-variant-numeric: tabular-nums; margin: 1.5rem 0; }
    p.hint { margin-top: 2rem; font-size: 0.85rem; opacity: 0.7; }
    p.meta { font-size: 0.8rem; opacity: 0.55; }
  </style>
</head>
<body>
  <h1>Ticker</h1>
  <p class="count">{{.Count}}</p>
  <p class="meta">seconds since start (server clock)</p>
  <p class="hint">
    A Go goroutine adds 1 every second. This page only reads the value
    (server-rendered template; auto-refresh each second).
  </p>
</body>
</html>
`))

func newTicker(count *atomic.Int64) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := page.Execute(w, map[string]int64{"Count": count.Load()}); err != nil {
			slog.Error("ticker template", "err", err)
		}
	})
	return mux
}
