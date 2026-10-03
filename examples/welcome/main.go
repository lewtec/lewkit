// Welcome is a page. A button opens the folder dialog.
//
//	go run ./cmd/lewkit release run --config ./examples/welcome/eletrocromo.json
//
// Android:
//
//	go run ./cmd/lewkit release run --config ./examples/welcome/eletrocromo.json --goos android --app --cgo
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/ui/gui"
)

//go:generate go tool templ generate

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	dirs, err := gui.Recent()
	if err != nil {
		dirs = nil
	}
	return app.App{
		Title: "Welcome",
		Handler: app.Web(newWelcome(ctx, welcomeConfig{
			title:    release.Name(),
			recent:   dirs,
			remember: gui.Remember,
		})),
	}.Run(ctx)
}

type chooseFunc func(context.Context, filedialog.Request) ([]string, error)

type openFunc func(...string) (fs.FS, error)

type rememberFunc func(string) error

type welcomeConfig struct {
	title    string
	recent   []gui.Directory
	choose   chooseFunc
	open     openFunc
	remember rememberFunc
}

func newWelcome(ctx context.Context, cfg welcomeConfig) http.Handler {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.title == "" {
		cfg.title = "Welcome"
	}
	if cfg.choose == nil {
		cfg.choose = filedialog.Choose
	}
	if cfg.open == nil {
		cfg.open = filedialog.Open
	}
	page := &welcomePage{
		ctx:      ctx,
		title:    cfg.title,
		recent:   append([]gui.Directory(nil), cfg.recent...),
		choose:   cfg.choose,
		open:     cfg.open,
		remember: cfg.remember,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", page.post)
	mux.HandleFunc("GET /{$}", page.get)
	return asset.Mount(mux)
}

// welcomePage keeps the last folder. The picker runs apart from the POST so a
// paused WebView does not cancel it before the user answers.
type welcomePage struct {
	ctx      context.Context
	title    string
	recent   []gui.Directory
	choose   chooseFunc
	open     openFunc
	remember rememberFunc

	mu      sync.Mutex
	pending bool
	picked  string
	names   []string
	note    string
}

type welcomeLink struct {
	Label string
	Href  string
}

type welcomeView struct {
	Title   string
	Pending bool
	Picked  string
	Names   []string
	Note    string
	Recent  []welcomeLink
}

func (p *welcomePage) post(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	if p.pending {
		p.mu.Unlock()
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	p.pending = true
	p.picked = ""
	p.names = nil
	p.note = ""
	p.mu.Unlock()
	go p.pick()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *welcomePage) pick() {
	paths, err := p.choose(p.ctx, filedialog.Request{Title: "Open folder", Folder: true})
	p.finish(paths, err)
}

func (p *welcomePage) finish(paths []string, err error) {
	var names []string
	picked := ""
	note := ""
	if err != nil {
		if !errors.Is(err, filedialog.ErrCanceled) {
			if errors.Is(err, driver.ErrUnavailable) {
				note = "No folder dialog on this system."
			} else {
				note = err.Error()
			}
		}
	} else if len(paths) > 0 {
		picked = paths[0]
		names, err = p.list(picked)
		if err != nil {
			note = err.Error()
		}
		if p.remember != nil && !strings.HasPrefix(picked, "content:") {
			if err := p.remember(picked); err != nil {
				slog.Error("welcome remember", "err", err)
			}
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = false
	p.picked = picked
	p.names = names
	p.note = note
}

func (p *welcomePage) list(dir string) ([]string, error) {
	tree, err := p.open(dir)
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(tree, ".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func (p *welcomePage) get(w http.ResponseWriter, r *http.Request) {
	if dir := r.URL.Query().Get("path"); dir != "" {
		p.openPath(dir)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := welcomeDocument(p.view()).Render(r.Context(), w); err != nil {
		slog.Error("welcome template", "err", err)
	}
}

func (p *welcomePage) openPath(dir string) {
	p.mu.Lock()
	if p.pending {
		p.mu.Unlock()
		return
	}
	p.pending = true
	p.mu.Unlock()
	p.finish([]string{dir}, nil)
}

func (p *welcomePage) view() welcomeView {
	p.mu.Lock()
	defer p.mu.Unlock()
	view := welcomeView{
		Title:   p.title,
		Pending: p.pending,
		Picked:  p.picked,
		Names:   append([]string(nil), p.names...),
		Note:    p.note,
	}
	if p.pending {
		return view
	}
	for _, dir := range p.recent {
		view.Recent = append(view.Recent, welcomeLink{
			Label: dir.Path,
			Href:  "/?" + url.Values{"path": {dir.Path}}.Encode(),
		})
	}
	return view
}
