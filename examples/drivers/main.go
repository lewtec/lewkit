package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/lewtec/lewkit/cmd/lewkit/experiments"
	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

//go:generate go tool templ generate

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Drivers",
		Handler: app.Web(newPage(ctx)),
	}.Run(ctx)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = Home(driver.Doctor(r.Context())).Render(r.Context(), w)
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
