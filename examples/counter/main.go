// Counter is a server-rendered increment form.
//
//	go run ./cmd/lewkit release run --config ./examples/counter/eletrocromo.json
package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Counter",
		Handler: app.Web(newCounter()),
	}.Run(ctx)
}

var page = template.Must(template.New("counter").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Counter</title>
  <style>
    :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
    body { max-width: 28rem; margin: 3rem auto; padding: 0 1rem; text-align: center; }
    h1 { font-size: 1.25rem; font-weight: 600; }
    .count { font-size: 4rem; font-variant-numeric: tabular-nums; margin: 1.5rem 0; }
    form { display: inline-flex; gap: 0.75rem; }
    button {
      font: inherit; padding: 0.5rem 1.25rem; border-radius: 0.5rem;
      border: 1px solid color-mix(in srgb, CanvasText 25%, transparent);
      background: color-mix(in srgb, CanvasText 8%, Canvas); cursor: pointer;
    }
    button:hover { background: color-mix(in srgb, CanvasText 14%, Canvas); }
    p.hint { margin-top: 2rem; font-size: 0.85rem; opacity: 0.7; }
  </style>
</head>
<body>
  <h1>Counter</h1>
  <p class="count">{{.Count}}</p>
  <form method="POST" action="/">
    <button type="submit" name="op" value="dec">−</button>
    <button type="submit" name="op" value="inc">+</button>
  </form>
  <form method="POST" action="/" style="display:block;margin-top:0.75rem">
    <button type="submit" name="op" value="reset">reset</button>
  </form>
  <p class="hint">Server-rendered with Go html/template. Close with Ctrl+C in the terminal.</p>
</body>
</html>
`))

func newCounter() http.Handler {
	var count atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		switch r.Form.Get("op") {
		case "inc":
			count.Add(1)
		case "dec":
			count.Add(-1)
		case "reset":
			count.Store(0)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, map[string]int64{"Count": count.Load()}); err != nil {
			slog.Error("counter template", "err", err)
		}
	})
	return mux
}
