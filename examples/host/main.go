// Command host reads the host drivers, changes brightness, and copies text.
package main

import (
	"context"
	"html/template"
	"net/http"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/lewtec/lewkit/x/release"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	return app.App{
		Title:   "Host",
		Handler: app.Web(newMux()),
	}.Run(ctx)
}

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", home)
	mux.HandleFunc("POST /act", postAct)
	return asset.Mount(mux)
}

type page struct {
	Style  string
	Notice string
	Sample string
	Rows   []result
}

func home(w http.ResponseWriter, r *http.Request) {
	id, err := release.AppID()
	if err != nil {
		id = ""
	}
	data := page{
		Style:  sakuracss.Path,
		Notice: r.URL.Query().Get("notice"),
		Sample: release.Name(),
		Rows:   probe(r.Context(), id),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func postAct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := action{
		Name: r.PostForm.Get("action"),
		Text: r.PostForm.Get("text"),
	}.run(r.Context())
	notice := "ok"
	if err != nil {
		notice = err.Error()
	}
	http.Redirect(w, r, "/?notice="+template.URLQueryEscaper(notice), http.StatusSeeOther)
}

var pageTmpl = template.Must(template.New("host").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Host</title>
<link rel="stylesheet" href="{{ .Style }}">
</head>
<body>
<h1>Host</h1>
{{ if .Notice }}<p>{{ .Notice }}</p>{{ end }}
<dl>
{{ range .Rows }}<dt>{{ .Name }}</dt><dd>{{ .Text }}</dd>{{ end }}
</dl>
<form method="post" action="/act">
<button name="action" value="brightness-up">Brightness up</button>
<button name="action" value="brightness-down">Brightness down</button>
</form>
<form method="post" action="/act">
<input name="text" value="{{ .Sample }}">
<button name="action" value="clipboard">Copy text</button>
</form>
</body>
</html>
`))
