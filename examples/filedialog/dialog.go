// Files is a page. A button opens the file dialog and lists the files.
//
//	go run ./cmd/lewkit release run --config ./examples/filedialog/eletrocromo.json
//
// Android (document picker, needs a device and the NDK):
//
//	go run ./cmd/lewkit release run --config ./examples/filedialog/eletrocromo.json --goos android --app --cgo
package main

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
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
		Handler: app.Web(newFiles(ctx, nil, nil)),
	}.Run(ctx)
}

type chooseFunc func(context.Context, filedialog.Request) ([]string, error)

type openFunc func(...string) (fs.FS, error)

func newFiles(ctx context.Context, choose chooseFunc, open openFunc) http.Handler {
	if ctx == nil {
		ctx = context.Background()
	}
	if choose == nil {
		choose = filedialog.Choose
	}
	if open == nil {
		open = filedialog.Open
	}
	page := &filesPage{ctx: ctx, choose: choose, open: open}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", page.post)
	mux.HandleFunc("GET /{$}", page.get)
	mux.HandleFunc("GET /file", page.file)
	return asset.Mount(mux)
}

// filesPage keeps the last pick. The picker runs apart from the POST so a
// paused WebView does not cancel it before the user answers.
type filesPage struct {
	ctx    context.Context
	choose chooseFunc
	open   openFunc

	mu      sync.Mutex
	pending bool
	fsys    fs.FS
	note    string
}

type filesEntry struct {
	Name string
	Href string
	Dir  bool
	Size string
}

type filesView struct {
	Pending bool
	Here    string
	Up      string
	Entries []filesEntry
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
	p.fsys = nil
	p.note = ""
	p.mu.Unlock()
	go p.pick(req)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *filesPage) pick(req filedialog.Request) {
	paths, err := p.choose(p.ctx, req)
	var fsys fs.FS
	if err == nil {
		fsys, err = p.open(paths...)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = false
	if err != nil {
		p.fsys = nil
		p.note = err.Error()
		return
	}
	p.fsys = fsys
	p.note = ""
}

func (p *filesPage) get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := filesDocument(p.view(r.URL.Query().Get("path"))).Render(r.Context(), w); err != nil {
		slog.Error("files template", "err", err)
	}
}

func (p *filesPage) file(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("path")
	fsys := p.current()
	if fsys == nil || name == "" {
		http.NotFound(w, r)
		return
	}
	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	ctype := mime.TypeByExtension(path.Ext(st.Name()))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("files read", "err", err)
	}
}

func (p *filesPage) current() fs.FS {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fsys
}

func (p *filesPage) view(raw string) filesView {
	p.mu.Lock()
	fsys := p.fsys
	view := filesView{Pending: p.pending, Note: p.note}
	p.mu.Unlock()
	if fsys == nil || view.Pending {
		return view
	}
	name := raw
	if name == "" {
		name = "."
	}
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		view.Note = err.Error()
		return view
	}
	if name != "." {
		view.Here = name
		parent := path.Dir(name)
		if parent == "." {
			view.Up = "/"
		} else {
			view.Up = "/?" + url.Values{"path": {parent}}.Encode()
		}
	}
	view.Entries = make([]filesEntry, 0, len(entries))
	for _, entry := range entries {
		full := entry.Name()
		if name != "." {
			full = path.Join(name, entry.Name())
		}
		item := filesEntry{
			Name: entry.Name(),
			Dir:  entry.IsDir(),
			Href: entryHref(entry.IsDir(), full),
		}
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				item.Size = formatSize(info.Size())
			}
		}
		view.Entries = append(view.Entries, item)
	}
	return view
}

func entryHref(dir bool, full string) string {
	q := url.Values{"path": {full}}.Encode()
	if dir {
		return "/?" + q
	}
	return "/file?" + q
}

func formatSize(n int64) string {
	if n < 0 {
		n = 0
	}
	if n < 1024 {
		return strconv.FormatInt(n, 10) + " B"
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(n)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	return strconv.FormatFloat(value, 'f', 1, 64) + " " + units[i]
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
