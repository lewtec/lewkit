package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"sync"

	"github.com/lewtec/lewkit/cmd/lewkit/experiments"
	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

func main() {
	entry.Main(func(ctx context.Context) error {
		return app.App{
			Title:   "Drivers",
			Handler: app.Web(newPage(ctx)),
		}.Run(ctx)
	})
}

func newPage(ctx context.Context) http.Handler {
	page := &page{ctx: ctx}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", page.home)
	mux.HandleFunc("POST /triangle", page.openTriangle)
	return mux
}

type page struct {
	ctx  context.Context
	mu   sync.Mutex
	open bool
}

func (p *page) home(w http.ResponseWriter, r *http.Request) {
	report := driver.Doctor(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Drivers</title>
<style>
body { font: 16px/1.4 sans-serif; margin: 0; padding: 16px 16px 88px; }
h1 { font-size: 1.4rem; }
section { margin: 1.2rem 0; }
h2 { font-size: 1rem; margin-bottom: 0.3rem; }
ul { list-style: none; padding: 0; margin: 0; }
li { padding: 0.35rem 0; border-bottom: 1px solid #ddd; }
.ok { color: #0a0; }
.no { color: #888; }
.bar { position: fixed; left: 0; right: 0; bottom: 0; padding: 12px 16px; background: #111; }
button, #triangle { display: block; box-sizing: border-box; width: 100%%; font: inherit; padding: 12px; text-align: center; color: inherit; background: #fff; text-decoration: none; }
#status { color: #fff; margin: 0 0 8px; min-height: 1.2em; }
</style>
</head>
<body>
<h1>Drivers</h1>
`)
	for _, iface := range report {
		fmt.Fprintf(w, "<section><h2>%s</h2><ul>", html.EscapeString(iface.Name))
		if len(iface.Drivers) == 0 {
			fmt.Fprint(w, "<li>none</li>")
		}
		for _, d := range iface.Drivers {
			mark := "no"
			label := "unavailable"
			if d.Available {
				mark = "ok"
				label = "available"
			}
			if d.Selected {
				label = "selected"
			}
			fmt.Fprintf(w, `<li><span class="%s">%s</span> %s <small>%s · %d</small>`,
				mark, html.EscapeString(label), html.EscapeString(d.Name), html.EscapeString(d.ID), d.Weight)
			if d.Error != nil {
				fmt.Fprintf(w, "<br><small>%s</small>", html.EscapeString(d.Error.Error()))
			}
			fmt.Fprint(w, "</li>")
		}
		fmt.Fprint(w, "</ul></section>")
	}
	fmt.Fprint(w, `<div class="bar"><p id="status"></p><button type="button" id="triangle">Open triangle</button></div>
<script>
document.getElementById("triangle").onclick = async () => {
  const status = document.getElementById("status");
  status.textContent = "opening…";
  const res = await fetch("/triangle", { method: "POST" });
  const body = await res.json();
  status.textContent = body.error || "opened";
};
</script>
</body></html>
`)
}

func (p *page) openTriangle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	model, err := experiments.TriangleModel(800, 600)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	p.mu.Lock()
	if p.open {
		p.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		return
	}
	p.open = true
	p.mu.Unlock()
	if err := app.Open(p.ctx, app.GUI(model), "Triangle", 800, 600); err != nil {
		p.mu.Lock()
		p.open = false
		p.mu.Unlock()
		slog.Error("triangle", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
