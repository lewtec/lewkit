package decompview

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/decomp"
	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
)

// Handler serves the preview of project. The database is read-only.
func Handler(project *decomp.Project) http.Handler {
	view := &preview{project: project}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", view.index)
	mux.HandleFunc("GET /artifact/{name}", view.artifact)
	mux.HandleFunc("GET /artifact/{name}/function/{address}", view.function)
	return asset.Mount(mux)
}

// preview reads one project. The project database is not safe for
// overlapping calls, so every page takes the same lock.
type preview struct {
	project *decomp.Project
	mu      sync.Mutex
}

type previewPage struct {
	Title       string
	Style       string
	Project     string
	ProjectName string
	Heading     string
	Notice      string
	Summary     string
	Signature   string
	Comment     string
	Listing     string
	Source      string
	ShowSource  bool
	Rows        []previewRow
	Items       []paletteItem
}

type previewRow struct {
	Label  string
	Href   string
	Detail string
}

type paletteItem struct {
	Group    string
	Label    string
	Href     string
	Keywords string
}

func (view *preview) index(response http.ResponseWriter, _ *http.Request) {
	view.mu.Lock()
	page := view.indexPage()
	view.mu.Unlock()
	writePage(response, page)
}

func (view *preview) artifact(response http.ResponseWriter, request *http.Request) {
	view.mu.Lock()
	page, ok := view.artifactPage(request.PathValue("name"))
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	writePage(response, page)
}

func (view *preview) function(response http.ResponseWriter, request *http.Request) {
	address, err := parseAddress(request.PathValue("address"))
	if err != nil {
		http.Error(response, "bad function address", http.StatusBadRequest)
		return
	}
	view.mu.Lock()
	page, ok := view.functionPage(request.PathValue("name"), address)
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	writePage(response, page)
}

func (view *preview) indexPage() previewPage {
	page := view.basePage("Artifacts")
	for _, artifact := range view.project.Artifacts() {
		page.Rows = append(page.Rows, previewRow{
			Label:  artifact.Name(),
			Href:   artifactPath(artifact.Name()),
			Detail: fmt.Sprintf("%s, %d functions", artifact.Format(), len(artifact.Functions())),
		})
	}
	if len(page.Rows) == 0 {
		page.Notice = "No artifacts."
	}
	return page
}

func (view *preview) artifactPage(name string) (previewPage, bool) {
	artifact, ok := view.project.Artifact(name)
	if !ok {
		return previewPage{}, false
	}
	page := view.basePage(artifact.Name())
	page.Summary = artifact.Summary()
	for _, function := range artifact.Functions() {
		page.Rows = append(page.Rows, previewRow{
			Label:  function.Name,
			Href:   functionPath(name, function.Address),
			Detail: formatHex(function.Address),
		})
	}
	return page, true
}

func (view *preview) functionPage(name string, address uint64) (previewPage, bool) {
	artifact, ok := view.project.Artifact(name)
	if !ok {
		return previewPage{}, false
	}
	function, ok := artifact.Function(address)
	if !ok {
		return previewPage{}, false
	}
	listing, err := artifact.Listing(address)
	if err != nil {
		return previewPage{}, false
	}
	page := view.basePage(function.Name)
	page.Signature = function.Signature
	page.Comment = function.Comment
	page.Listing = listing
	page.Source = function.Source
	page.ShowSource = true
	return page, true
}

func (view *preview) basePage(heading string) previewPage {
	path := view.project.Path()
	return previewPage{
		Title:       "decompview",
		Style:       sakuracss.Path,
		Project:     path,
		ProjectName: filepath.Base(path),
		Heading:     heading,
		Items:       view.paletteItems(),
	}
}

func (view *preview) paletteItems() []paletteItem {
	items := []paletteItem{{
		Group:    "Project",
		Label:    "Artifacts",
		Href:     "/",
		Keywords: paletteKeywords("Project", "Artifacts", view.project.Path()),
	}}
	for _, artifact := range view.project.Artifacts() {
		items = append(items, paletteItem{
			Group:    "Artifact",
			Label:    artifact.Name(),
			Href:     artifactPath(artifact.Name()),
			Keywords: paletteKeywords("Artifact", artifact.Name(), artifact.Format()),
		})
		for _, function := range artifact.Functions() {
			label := function.Name + " " + formatHex(function.Address)
			items = append(items, paletteItem{
				Group:    artifact.Name(),
				Label:    label,
				Href:     functionPath(artifact.Name(), function.Address),
				Keywords: paletteKeywords(artifact.Name(), function.Name, function.Signature, formatHex(function.Address), strconv.FormatUint(function.Address, 16)),
			})
		}
	}
	return items
}

func writePage(response http.ResponseWriter, page previewPage) {
	var body bytes.Buffer
	if err := pageTemplate.Execute(&body, page); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = response.Write(body.Bytes())
}

func artifactPath(name string) string {
	return "/artifact/" + url.PathEscape(name)
}

func functionPath(name string, address uint64) string {
	return artifactPath(name) + "/function/" + formatHex(address)
}

func formatHex(address uint64) string {
	return fmt.Sprintf("0x%x", address)
}

func parseAddress(text string) (uint64, error) {
	text = strings.TrimPrefix(text, "0x")
	text = strings.TrimPrefix(text, "0X")
	if text == "" {
		return 0, fmt.Errorf("empty function address")
	}
	address, err := strconv.ParseUint(text, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("function address: %w", err)
	}
	return address, nil
}

func paletteKeywords(parts ...string) string {
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(strings.ToLower(part))
	}
	text := builder.String()
	if strings.Contains(text, ":") {
		text += " " + strings.ReplaceAll(text, ":", " ")
	}
	return text
}

var pageTemplate = template.Must(template.New("preview").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{ .Title }}</title>
<link rel="stylesheet" href="{{ .Style }}">
<style>
header { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
header p { margin: 0; }
#palette { width: min(36rem, calc(100vw - 2rem)); padding: 0; border: 1px solid; }
#palette input { width: 100%; box-sizing: border-box; margin: 0; border-width: 0 0 1px; }
#palette-list { list-style: none; margin: 0; padding: 0; max-height: 24rem; overflow: auto; }
#palette-list a { display: block; padding: 0.35rem 0.6rem; text-decoration: none; }
#palette-list .palette-group { display: block; font-size: 0.75rem; opacity: 0.7; }
#palette-list .palette-active { background: #f4e4b3; }
pre { overflow: auto; }
table { width: 100%; }
</style>
</head>
<body>
<header>
<p><a href="/" title="{{ .Project }}">{{ .ProjectName }}</a></p>
<button id="palette-open" type="button">Jump <kbd id="palette-hotkey">Ctrl+K</kbd></button>
</header>
<h1>{{ .Heading }}</h1>
{{ if .Notice }}<p>{{ .Notice }}</p>{{ end }}
{{ if .Signature }}<p>{{ .Signature }}</p>{{ end }}
{{ if .Comment }}<p>{{ .Comment }}</p>{{ end }}
{{ if .Summary }}<pre>{{ .Summary }}</pre>{{ end }}
{{ if .Rows }}<table>
{{ range .Rows }}<tr><td><a href="{{ .Href }}">{{ .Label }}</a></td><td>{{ .Detail }}</td></tr>
{{ end }}</table>{{ end }}
{{ if .Listing }}<h2>Listing</h2><pre>{{ .Listing }}</pre>{{ end }}
{{ if .ShowSource }}<h2>Matched source</h2>{{ if .Source }}<pre>{{ .Source }}</pre>{{ else }}<p>No matched source.</p>{{ end }}{{ end }}
<dialog id="palette" data-cmdpalette aria-label="Jump to">
<input id="palette-input" type="text" placeholder="Jump to…" autocomplete="off" aria-controls="palette-list" aria-autocomplete="list">
<p id="palette-empty" hidden>No matches</p>
<ul id="palette-list" role="listbox">
{{ range .Items }}<li data-cmd-item data-keywords="{{ .Keywords }}" role="option"><a href="{{ .Href }}"><span class="palette-group">{{ .Group }}</span> {{ .Label }}</a></li>
{{ end }}</ul>
</dialog>
<script>
(function () {
  var dialog = document.getElementById("palette");
  var input = document.getElementById("palette-input");
  var list = document.getElementById("palette-list");
  var empty = document.getElementById("palette-empty");
  var openButton = document.getElementById("palette-open");
  var hotkey = document.getElementById("palette-hotkey");
  if (!dialog || !input || !list) return;
  var items = Array.prototype.slice.call(list.querySelectorAll("[data-cmd-item]"));
  var visible = items.slice();
  var active = 0;
  if (hotkey && /Mac|iPhone|iPad|iPod/.test(navigator.platform || "")) {
    hotkey.textContent = "\u2318K";
  }
  function setActive(index) {
    for (var i = 0; i < items.length; i++) {
      items[i].setAttribute("aria-selected", "false");
      items[i].classList.remove("palette-active");
    }
    if (visible.length === 0) {
      active = -1;
      return;
    }
    if (index < 0) index = visible.length - 1;
    if (index >= visible.length) index = 0;
    active = index;
    visible[active].setAttribute("aria-selected", "true");
    visible[active].classList.add("palette-active");
    visible[active].scrollIntoView({ block: "nearest" });
  }
  function filter() {
    var query = input.value.trim().toLowerCase();
    var tokens = query ? query.split(/\s+/).filter(Boolean) : [];
    visible = [];
    for (var i = 0; i < items.length; i++) {
      var haystack = (items[i].getAttribute("data-keywords") || "").toLowerCase();
      var matched = true;
      for (var token = 0; token < tokens.length; token++) {
        if (haystack.indexOf(tokens[token]) === -1) {
          matched = false;
          break;
        }
      }
      items[i].hidden = !matched;
      if (matched) visible.push(items[i]);
    }
    if (empty) empty.hidden = visible.length !== 0;
    setActive(0);
  }
  function openPalette() {
    if (dialog.open) return;
    input.value = "";
    filter();
    if (dialog.showModal) dialog.showModal();
    else dialog.setAttribute("open", "");
    input.focus();
  }
  function closePalette() {
    if (dialog.close && dialog.open) dialog.close();
    else dialog.removeAttribute("open");
  }
  function activate() {
    if (active < 0 || active >= visible.length) return;
    var link = visible[active].querySelector("a");
    if (link) window.location.href = link.href;
  }
  document.addEventListener("keydown", function (event) {
    var chord = (event.metaKey || event.ctrlKey) && (event.key === "k" || event.key === "K");
    if (chord) {
      event.preventDefault();
      if (dialog.open) closePalette();
      else openPalette();
      return;
    }
    if (!dialog.open) return;
    if (event.key === "Escape") {
      event.preventDefault();
      closePalette();
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive(active + 1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive(active - 1);
    } else if (event.key === "Enter") {
      event.preventDefault();
      activate();
    }
  }, true);
  input.addEventListener("input", filter);
  if (openButton) openButton.addEventListener("click", function (event) {
    event.preventDefault();
    openPalette();
  });
  dialog.addEventListener("click", function (event) {
    if (event.target === dialog) closePalette();
  });
  list.addEventListener("mousemove", function (event) {
    var item = event.target.closest("[data-cmd-item]");
    if (!item || item.hidden) return;
    var index = visible.indexOf(item);
    if (index >= 0) setActive(index);
  });
  filter();
})();
</script>
</body>
</html>
`))
