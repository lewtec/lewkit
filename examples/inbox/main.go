// Inbox shows URL and file opens, and writes notes under the app dirs.
//
//	go run ./cmd/lewkit release run --config ./examples/inbox/eletrocromo.json
//	./examples/inbox/sim.sh sim
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/lewtec/lewkit/x/driver/share"
	_ "github.com/lewtec/lewkit/x/driver/share/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/release"
)

//go:generate go tool templ generate

const (
	packageID = "br.tec.lew.eletrocromo.inbox"
	scheme    = "eletrocromo-inbox"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	st := &state{ctx: ctx}
	go func() {
		err := Listen(ctx, appID(), st.onOpen)
		if err != nil && ctx.Err() == nil {
			slog.Error("open", "err", err)
		}
	}()
	return app.App{
		Title:   "Inbox",
		Handler: app.Web(newInbox(st)),
	}.Run(ctx)
}

func appID() string {
	id, err := release.AppID()
	if err != nil || id == "" {
		return packageID
	}
	return id
}

type event struct {
	When time.Time
	Kind string
	Text string
	Body string
}

type pageView struct {
	Scheme string
	Dirs   dirs.Dirs
	Events []event
	Names  []string
	Probes []urlProbe
}

type state struct {
	ctx    context.Context
	mu     sync.Mutex
	events []event
}

func (s *state) add(ev event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append([]event{ev}, s.events...)
	if len(s.events) > 20 {
		s.events = s.events[:20]
	}
}

func (s *state) list() []event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *state) onOpen(ev Event) error {
	item := event{When: time.Now()}
	switch ev.Kind {
	case KindURL:
		if name := inboxNameFromURL(ev.URL); name != "" {
			if path, ok := inboxFile(s.ctx, name); ok {
				item.Kind = "url"
				item.Text = name
				item.Body = describeURL(ev.URL) + "\n" + path
				s.add(item)
				return nil
			}
		}
		item.Kind = "url"
		item.Text = ev.URL
		item.Body = describeURL(ev.URL)
	case KindFiles:
		item.Kind = "files"
		item.Text = strings.Join(ev.Paths, ", ")
		if len(ev.Paths) > 0 && looksText(ev.Paths[0]) {
			item.Body = readHead(ev.Paths[0], 2048)
		}
	default:
		return nil
	}
	s.add(item)
	return nil
}

func inboxNameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	for _, key := range []string{"name", "file", "path"} {
		if n := sanitizeName(u.Query().Get(key)); n != "" {
			return n
		}
	}
	if u.Host == "inbox" {
		return sanitizeName(strings.TrimPrefix(u.Path, "/"))
	}
	if rest, ok := strings.CutPrefix(strings.TrimPrefix(u.Path, "/"), "inbox/"); ok {
		return sanitizeName(rest)
	}
	if u.Opaque != "" {
		if rest, ok := strings.CutPrefix(u.Opaque, "inbox/"); ok {
			return sanitizeName(rest)
		}
	}
	return ""
}

func sanitizeName(n string) string {
	n = filepath.Base(strings.TrimSpace(n))
	if n == "." || n == ".." || n == "" {
		return ""
	}
	return n
}

func inboxFile(ctx context.Context, name string) (string, bool) {
	d, err := dirs.Resolve(ctx, appID())
	if err != nil {
		return "", false
	}
	path := filepath.Join(d.Inbox, name)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

func describeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "parse: " + err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "scheme=%s", u.Scheme)
	if u.Opaque != "" {
		fmt.Fprintf(&b, " opaque=%s", u.Opaque)
	}
	if u.Host != "" {
		fmt.Fprintf(&b, " host=%s", u.Host)
	}
	if u.Path != "" {
		fmt.Fprintf(&b, " path=%s", u.Path)
	}
	if u.RawQuery != "" {
		fmt.Fprintf(&b, " query=%s", u.RawQuery)
	}
	if u.Fragment != "" {
		fmt.Fprintf(&b, " frag=%s", u.Fragment)
	}
	return b.String()
}

func looksText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".json", ".csv", ".xml", ".html", ".htm", ".go":
		return true
	default:
		return false
	}
}

type urlProbe struct {
	Href  string
	Label string
}

func urlProbes(scheme string, names []string) []urlProbe {
	out := []urlProbe{
		{scheme + "://from-webview", "from-webview"},
		{scheme + "://", "empty host"},
		{scheme + "://x/y?q=1&empty=&sp=a+b#frag", "query+frag"},
		{scheme + "://caf%C3%A9/%E6%97%A5%E6%9C%AC%E8%AA%9E", "unicode"},
		{scheme + ":opaque-rest", "opaque"},
		{scheme + "://inbox?name=missing.bin", "missing name"},
	}
	if len(names) > 0 {
		out = append(out, urlProbe{
			Href:  scheme + "://inbox?name=" + url.QueryEscape(names[0]),
			Label: "name=" + names[0],
		})
	}
	return out
}

func readHead(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	buf := make([]byte, n+1)
	got, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err.Error()
	}
	text := string(buf[:min(got, n)])
	if got > n {
		text += "…"
	}
	return text
}

func inboxNames(inbox string) []string {
	ents, err := os.ReadDir(inbox)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func newInbox(st *state) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /note", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(r.Form.Get("text"))
		if text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		d, err := dirs.Resolve(r.Context(), appID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		notes := filepath.Join(d.Data, "notes")
		if err := os.MkdirAll(notes, 0o700); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		name := time.Now().Format("20060102-150405") + ".txt"
		if err := os.WriteFile(filepath.Join(notes, name), []byte(text+"\n"), 0o600); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "note", Text: name, Body: text})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /share", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(r.Form.Get("text"))
		if text == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err := share.Out(r.Context(), share.Item{Text: text}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "share", Text: text})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /share-file", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		name := filepath.Base(strings.TrimSpace(r.Form.Get("name")))
		if name == "." || name == ".." || name == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		d, err := dirs.Resolve(r.Context(), appID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path := filepath.Join(d.Inbox, name)
		if err := share.Out(r.Context(), share.Item{Paths: []string{path}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		st.add(event{When: time.Now(), Kind: "share", Text: name})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		raw := strings.TrimSpace(r.Form.Get("url"))
		if raw == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		ev, ok := Token(raw)
		if !ok {
			ev = Event{Kind: KindURL, URL: raw}
		}
		if err := st.onOpen(ev); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		d, err := dirs.Resolve(r.Context(), appID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		names := inboxNames(d.Inbox)
		view := pageView{
			Scheme: scheme,
			Dirs:   d,
			Events: st.list(),
			Names:  names,
			Probes: urlProbes(scheme, names),
		}
		if err := page(view).Render(r.Context(), w); err != nil {
			slog.Error("inbox template", "err", err)
		}
	})
	return mux
}
