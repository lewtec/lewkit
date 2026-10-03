// Files is a page. A button opens the file dialog and lists each path.
//
//	go run ./cmd/lewkit release run --config ./examples/filedialog/eletrocromo.json
//
// Android (document picker, needs a device and the NDK):
//
//	go run ./cmd/lewkit release run --config ./examples/filedialog/eletrocromo.json --goos android --app --cgo
package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/asset"
)

//go:generate go tool templ generate

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	return app.App{
		Title:   "Files",
		Handler: app.Web(newFiles(ctx, nil)),
	}.Run(ctx)
}

type chooseFunc func(context.Context, filedialog.Request) ([]string, error)

func newFiles(ctx context.Context, choose chooseFunc) http.Handler {
	if ctx == nil {
		ctx = context.Background()
	}
	if choose == nil {
		choose = filedialog.Choose
	}
	page := &filesPage{ctx: ctx, choose: choose}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", page.post)
	mux.HandleFunc("GET /{$}", page.get)
	return asset.Mount(mux)
}

// filesPage keeps the last pick. The picker runs apart from the POST so a
// paused WebView does not cancel it before the user answers.
type filesPage struct {
	ctx    context.Context
	choose chooseFunc

	mu      sync.Mutex
	pending bool
	paths   []string
	note    string
}

type filesView struct {
	Pending bool
	Paths   []string
	Note    string
}

func (p *filesPage) post(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	req, ok := requestOf(r.Form.Get("op"), r.Form.Get("name"))
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	p.mu.Lock()
	if p.pending {
		p.mu.Unlock()
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	p.pending = true
	p.paths = nil
	p.note = ""
	p.mu.Unlock()
	go p.pick(req)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *filesPage) pick(req filedialog.Request) {
	paths, err := p.choose(p.ctx, req)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = false
	if err != nil {
		p.paths = nil
		p.note = err.Error()
		return
	}
	p.paths = paths
	p.note = ""
}

func (p *filesPage) get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := filesDocument(p.snapshot()).Render(r.Context(), w); err != nil {
		slog.Error("files template", "err", err)
	}
}

func (p *filesPage) snapshot() filesView {
	p.mu.Lock()
	defer p.mu.Unlock()
	return filesView{
		Pending: p.pending,
		Paths:   append([]string(nil), p.paths...),
		Note:    p.note,
	}
}

func requestOf(op, name string) (filedialog.Request, bool) {
	switch op {
	case "files":
		return filedialog.Request{Title: "Choose files", Multiple: true}, true
	case "folder":
		return filedialog.Request{Title: "Choose folder", Folder: true}, true
	case "save":
		return filedialog.Request{Title: "Save", Name: name, Save: true}, true
	default:
		return filedialog.Request{}, false
	}
}
