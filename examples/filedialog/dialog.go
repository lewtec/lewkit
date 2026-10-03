// Files is a page. A button opens the file dialog and shows the files.
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
	"strings"
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
	mux.HandleFunc("GET /thumb", page.thumb)
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
	Name  string
	Href  string
	Dir   bool
	Size  string
	Media string
	Src   string
	Text  string
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

func (p *filesPage) thumb(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("path")
	fsys := p.current()
	preview, ok := fsys.(interface{ Preview(string) (fs.File, error) })
	if fsys == nil || name == "" || !ok {
		p.file(w, r)
		return
	}
	f, err := preview.Preview(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/jpeg")
	if _, err := io.Copy(w, f); err != nil {
		slog.Error("files thumb", "err", err)
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
		item := filesEntry{Name: entry.Name(), Dir: entry.IsDir()}
		if entry.IsDir() {
			item.Href = "/?" + url.Values{"path": {full}}.Encode()
		} else {
			info, err := entry.Info()
			if err == nil {
				item.Size = formatSize(info.Size())
			}
			shown, err := showFile(fsys, full)
			if err != nil {
				view.Note = err.Error()
			} else {
				item.Media = shown.Media
				item.Src = shown.Src
				item.Text = shown.Text
			}
		}
		view.Entries = append(view.Entries, item)
	}
	return view
}

// previewBytes is how much of a text file the page renders.
const previewBytes = 32 << 10

type shownFile struct {
	Media string
	Src   string
	Text  string
}

func showFile(fsys fs.FS, full string) (shownFile, error) {
	kind := mediaKind(full)
	src := "/file?" + url.Values{"path": {full}}.Encode()
	if kind == "image" {
		src = "/thumb?" + url.Values{"path": {full}}.Encode()
	}
	switch kind {
	case "image", "audio", "video":
		return shownFile{Media: kind, Src: src}, nil
	case "text":
		f, err := fsys.Open(full)
		if err != nil {
			return shownFile{}, err
		}
		defer f.Close()
		body, err := io.ReadAll(io.LimitReader(f, previewBytes+1))
		if err != nil {
			return shownFile{}, err
		}
		if len(body) > previewBytes {
			body = body[:previewBytes]
		}
		return shownFile{Media: "text", Text: string(body)}, nil
	default:
		return shownFile{Src: src}, nil
	}
}

func mediaKind(name string) string {
	media, _, err := mime.ParseMediaType(mime.TypeByExtension(path.Ext(name)))
	if err != nil || media == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(media, "image/"):
		return "image"
	case strings.HasPrefix(media, "audio/"):
		return "audio"
	case strings.HasPrefix(media, "video/"):
		return "video"
	case strings.HasPrefix(media, "text/"):
		return "text"
	}
	switch media {
	case "application/json", "application/xml", "application/javascript", "application/xhtml+xml":
		return "text"
	default:
		return ""
	}
}

func formatSize(n int64) string {
	if n < 0 {
		n = 0
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(n)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	if i == 0 {
		return strconv.FormatInt(n, 10) + " B"
	}
	digits := 0
	if value < 10 {
		digits = 1
	}
	return strconv.FormatFloat(value, 'f', digits, 64) + " " + units[i]
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
