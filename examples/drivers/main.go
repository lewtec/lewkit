package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/examples/internal/scene"
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
	mux.HandleFunc("GET /{$}", page.home)
	mux.HandleFunc("GET /driver/{name}", page.driver)
	mux.HandleFunc("POST /driver/{name}/{op}", page.driver)
	mux.HandleFunc("POST /triangle", page.openTriangle)
	return mux
}

type page struct {
	ctx  context.Context
	mu   sync.Mutex
	open bool
	held held
}

func (p *page) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = Home(driver.Doctor(r.Context())).Render(r.Context(), w)
}

func (p *page) driver(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("name")
	iface, ok := interfaceBySlug(driver.Doctor(r.Context()), slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	spec, known := driverSpecs[slug]
	if r.Method == http.MethodPost {
		if !known || spec.run == nil {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		notice, err := spec.run(r.Context(), p, r.PathValue("op"), r)
		if errors.Is(err, errNoDriverOp) {
			http.NotFound(w, r)
			return
		}
		redirectResult(w, r, slug, notice, err)
		return
	}
	view := driverView{
		Slug:    slug,
		Name:    iface.Name,
		Drivers: iface.Drivers,
		Notice:  r.URL.Query().Get("notice"),
		Err:     r.URL.Query().Get("err"),
	}
	if !known || spec.load == nil {
		view.StateErr = errNoPanel.Error()
	} else {
		loaded, err := spec.load(r.Context(), p)
		view.Rows = loaded.rows
		view.Acts = loaded.acts
		if err != nil {
			view.StateErr = err.Error()
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := DriverPage(view).Render(r.Context(), w); err != nil {
		slog.Error("driver page", "err", err)
	}
}

func redirectResult(w http.ResponseWriter, r *http.Request, slug, notice string, err error) {
	query := url.Values{}
	if err != nil {
		query.Set("err", clip(err.Error(), 400))
	} else if notice != "" {
		query.Set("notice", clip(notice, 400))
	} else {
		query.Set("notice", "done")
	}
	http.Redirect(w, r, "/driver/"+url.PathEscape(slug)+"?"+query.Encode(), http.StatusSeeOther)
}

func clip(text string, n int) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n]) + "…"
}

func (p *page) openTriangle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	model, err := scene.TriangleModel(800, 600)
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
