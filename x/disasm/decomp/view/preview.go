package view

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/disasm/decomp"
	"github.com/lewtec/lewkit/x/http/asset"
)

var (
	ErrEmptyFunctionAddress = errors.New("empty function address")
	ErrFunctionAddress      = errors.New("function address")
	ErrServePreview         = errors.New("preview")
	ErrServeStopPreview     = errors.New("stop preview")
)

//go:generate go tool templ generate

// Handler serves the preview of project.
// Saving a function name, a signature, a held source, or an address note writes it through to the project.
func Handler(project *decomp.Project) http.Handler {
	view := &preview{project: project}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", view.index)
	mux.HandleFunc("GET /artifact/{name}/stream", view.stream)
	mux.HandleFunc("GET /artifact/{name}/at/{address}", view.address)
	mux.HandleFunc("GET /artifact/{name}", view.artifact)
	mux.HandleFunc("POST /artifact/{name}/note", view.saveNote)
	mux.HandleFunc("GET /artifact/{name}/function/{address}", view.function)
	mux.HandleFunc("POST /artifact/{name}/function/{address}/name", view.saveName)
	mux.HandleFunc("POST /artifact/{name}/function/{address}/signature", view.saveSignature)
	mux.HandleFunc("POST /artifact/{name}/function/{address}/source", view.saveSource)
	return asset.Mount(mux)
}

// preview reads one project. The project database is not safe for
// overlapping calls, so every page takes the same lock.
type preview struct {
	project *decomp.Project
	mu      sync.Mutex
}

type previewPage struct {
	Title           string
	Project         string
	ProjectName     string
	Heading         string
	Notice          string
	Summary         string
	Signature       string
	Comment         string
	StreamHref      string
	AssemblyHeading string
	Assembly        asmListing
	Source          string
	FunctionName    string
	AddressText     string
	AddressNote     string
	NameAction      string
	SignatureAction string
	SourceAction    string
	NoteAction      string
	Next            string
	Callees         []previewRow
	Notes           []previewNote
	Rows            []previewRow
	Items           []paletteItem
}

type previewNote struct {
	Address string
	Href    string
	Text    string
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

func (view *preview) index(response http.ResponseWriter, request *http.Request) {
	view.mu.Lock()
	page := view.indexPage()
	view.mu.Unlock()
	writePage(response, request, page)
}

func (view *preview) artifact(response http.ResponseWriter, request *http.Request) {
	view.namedPage(response, request, view.artifactPage)
}

func (view *preview) stream(response http.ResponseWriter, request *http.Request) {
	view.namedPage(response, request, view.streamPage)
}

func (view *preview) namedPage(response http.ResponseWriter, request *http.Request, page func(string) (previewPage, bool)) {
	view.mu.Lock()
	body, ok := page(request.PathValue("name"))
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	writePage(response, request, body)
}

func (view *preview) address(response http.ResponseWriter, request *http.Request) {
	address, err := parseAddress(request.PathValue("address"))
	if err != nil {
		http.Error(response, "bad address", http.StatusBadRequest)
		return
	}
	view.mu.Lock()
	page, ok := view.addressPage(request.PathValue("name"), address)
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	writePage(response, request, page)
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
	writePage(response, request, page)
}

func (view *preview) indexPage() previewPage {
	page := view.basePage("Artifacts")
	for _, artifact := range view.project.Artifacts() {
		page.Rows = append(page.Rows, previewRow{
			Label:  artifact.Slug(),
			Href:   artifactPath(artifact.Slug()),
			Detail: fmt.Sprintf("%s, %s, %d functions", artifactLabel(artifact), artifact.Format(), len(artifact.Functions())),
		})
	}
	if len(page.Rows) == 0 {
		page.Notice = "No artifacts."
	}
	return page
}

func (view *preview) artifactPage(name string) (previewPage, bool) {
	artifact, ok := view.project.Lookup(name)
	if !ok {
		return previewPage{}, false
	}
	slug := artifact.Slug()
	page := view.basePage(artifactLabel(artifact))
	page.Summary = artifact.Summary()
	page.StreamHref = artifactPath(slug) + "/stream"
	page.NoteAction = artifactPath(slug) + "/note"
	page.Next = artifactPath(slug)
	page.Notes = noteLinks(artifact, newLinker(artifact, artifact.Functions(), nil, false))
	for _, function := range artifact.Functions() {
		page.Rows = append(page.Rows, previewRow{
			Label:  function.Name,
			Href:   functionPath(slug, function.Address),
			Detail: fmt.Sprintf("%s %d", formatHex(function.Address), function.Size),
		})
	}
	return page, true
}

func (view *preview) streamPage(name string) (previewPage, bool) {
	artifact, ok := view.project.Lookup(name)
	if !ok {
		return previewPage{}, false
	}
	page := view.basePage(artifact.Slug())
	page.AssemblyHeading = "Instructions"
	page.Assembly = streamAssembly(artifact)
	page.Next = artifactPath(artifact.Slug())
	return page, true
}

func (view *preview) functionPage(name string, address uint64) (previewPage, bool) {
	artifact, ok := view.project.Lookup(name)
	if !ok {
		return previewPage{}, false
	}
	slug := artifact.Slug()
	function, ok := artifact.Function(address)
	if !ok {
		return previewPage{}, false
	}
	page := view.basePage(function.Name)
	page.Signature = function.Signature
	page.Comment = function.Comment
	page.AssemblyHeading = "Assembly"
	page.Assembly = functionAssembly(artifact, function)
	page.Source = function.Source
	page.FunctionName = function.Name
	page.AddressText = formatHex(function.Address)
	page.NameAction = functionPath(slug, function.Address) + "/name"
	page.SignatureAction = functionPath(slug, function.Address) + "/signature"
	page.SourceAction = functionPath(slug, function.Address) + "/source"
	page.NoteAction = artifactPath(slug) + "/note"
	page.Callees = callLinks(slug, function)
	page.Next = functionPath(slug, function.Address)
	if text, ok := artifact.Note(function.Address); ok {
		page.AddressNote = text
	}
	here := map[uint64]struct{}{}
	for _, instruction := range function.Instructions {
		here[instruction.Address] = struct{}{}
	}
	page.Notes = noteLinks(artifact, newLinker(artifact, artifact.Functions(), here, false))
	return page, true
}

func (view *preview) saveName(response http.ResponseWriter, request *http.Request) {
	view.saveFunction(response, request, func() string {
		return strings.TrimSpace(request.PostForm.Get("name"))
	}, (*decomp.Artifact).SetName)
}

func (view *preview) saveSignature(response http.ResponseWriter, request *http.Request) {
	view.saveFunction(response, request, func() string {
		return strings.TrimSpace(request.PostForm.Get("signature"))
	}, (*decomp.Artifact).SetSignature)
}

func (view *preview) saveSource(response http.ResponseWriter, request *http.Request) {
	view.saveFunction(response, request, func() string {
		source := request.PostForm.Get("source")
		if strings.TrimSpace(source) == "" {
			return ""
		}
		return source
	}, (*decomp.Artifact).SetSource)
}

func (view *preview) saveFunction(response http.ResponseWriter, request *http.Request, value func() string, save func(*decomp.Artifact, context.Context, uint64, string) error) {
	if err := request.ParseForm(); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	address, err := parseAddress(request.PathValue("address"))
	if err != nil {
		http.Error(response, "bad function address", http.StatusBadRequest)
		return
	}
	text := value()
	artifactName := request.PathValue("name")
	view.mu.Lock()
	artifact, ok := view.project.Lookup(artifactName)
	var saveErr error
	if ok {
		saveErr = save(artifact, request.Context(), address, text)
	}
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	if saveErr != nil {
		http.Error(response, saveErr.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(response, request, functionPath(artifact.Slug(), address), http.StatusSeeOther)
}

func (view *preview) saveNote(response http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	address, err := parseAddress(request.PostForm.Get("address"))
	if err != nil {
		http.Error(response, "bad note address", http.StatusBadRequest)
		return
	}
	artifactName := request.PathValue("name")
	text := strings.TrimSpace(request.PostForm.Get("text"))
	view.mu.Lock()
	artifact, ok := view.project.Lookup(artifactName)
	var saveErr error
	if ok {
		saveErr = artifact.SetNote(request.Context(), address, text)
	}
	view.mu.Unlock()
	if !ok {
		http.NotFound(response, request)
		return
	}
	if saveErr != nil {
		http.Error(response, saveErr.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(response, request, safeNext(artifact, request.PostForm.Get("next")), http.StatusSeeOther)
}

func (view *preview) addressPage(name string, address uint64) (previewPage, bool) {
	artifact, ok := view.project.Lookup(name)
	if !ok {
		return previewPage{}, false
	}
	section, inImage := artifact.Contains(address)
	if !inImage {
		return previewPage{}, false
	}
	link := newLinker(artifact, artifact.Functions(), nil, false)
	page := view.basePage(formatHex(address))
	if label, named := artifact.Label(address); named {
		page.Heading = formatHex(address) + " " + label
	}
	page.Notice = section
	page.Next = artifactPath(artifact.Slug())
	for _, reference := range artifact.XRefs(address) {
		href, _ := link.href(reference.From)
		page.Rows = append(page.Rows, previewRow{
			Label:  formatHex(reference.From),
			Href:   href,
			Detail: reference.Kind,
		})
	}
	if len(page.Rows) == 0 {
		page.Notice = section + ". No references."
	}
	return page, true
}

func noteLinks(artifact *decomp.Artifact, link *linker) []previewNote {
	out := make([]previewNote, 0, len(artifact.Notes()))
	for _, note := range artifact.Notes() {
		item := previewNote{Address: formatHex(note.Address), Text: note.Text}
		if href, ok := link.href(note.Address); ok {
			item.Href = href
		}
		out = append(out, item)
	}
	return out
}

func safeNext(artifact *decomp.Artifact, next string) string {
	for _, prefix := range []string{artifactPath(artifact.Slug()), artifactPath(artifact.Name())} {
		if next == prefix || strings.HasPrefix(next, prefix+"/") {
			return next
		}
	}
	return artifactPath(artifact.Slug())
}

func artifactLabel(artifact *decomp.Artifact) string {
	if artifact.Path() != "" {
		return filepath.Base(artifact.Path())
	}
	return artifact.Name()
}

func (view *preview) basePage(heading string) previewPage {
	path := view.project.Path()
	return previewPage{
		Title:       "view",
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
			Label:    artifact.Slug() + " " + artifactLabel(artifact),
			Href:     artifactPath(artifact.Slug()),
			Keywords: paletteKeywords("Artifact", artifact.Slug(), artifact.Digest(), artifact.Name(), artifactLabel(artifact), artifact.Format()),
		})
		for _, function := range artifact.Functions() {
			label := function.Name + " " + formatHex(function.Address)
			keywords := []string{artifact.Slug(), artifact.Name(), function.Name, function.Signature, formatHex(function.Address), strconv.FormatUint(function.Address, 16)}
			for _, note := range artifact.Notes() {
				if !noteBelongs(function, note.Address) {
					continue
				}
				keywords = append(keywords, note.Text, formatHex(note.Address))
			}
			items = append(items, paletteItem{
				Group:    artifact.Slug(),
				Label:    label,
				Href:     functionPath(artifact.Slug(), function.Address),
				Keywords: paletteKeywords(keywords...),
			})
		}
	}
	return items
}

func writePage(response http.ResponseWriter, request *http.Request, page previewPage) {
	if err := previewDocument(page).Render(request.Context(), response); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func artifactPath(name string) string {
	return "/artifact/" + url.PathEscape(name)
}

func functionPath(name string, address uint64) string {
	return artifactPath(name) + "/function/" + formatHex(address)
}

func formatHex(address uint64) string {
	return decomp.FormatAddress(address)
}

func parseAddress(text string) (uint64, error) {
	text = strings.TrimPrefix(text, "0x")
	text = strings.TrimPrefix(text, "0X")
	if text == "" {
		return 0, ErrEmptyFunctionAddress
	}
	address, err := strconv.ParseUint(text, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrFunctionAddress, err)
	}
	return address, nil
}

func callLinks(name string, function decomp.Function) []previewRow {
	seen := map[uint64]struct{}{}
	var links []previewRow
	for _, reference := range function.References {
		if reference.Kind != "call" {
			continue
		}
		if _, ok := seen[reference.To]; ok {
			continue
		}
		seen[reference.To] = struct{}{}
		links = append(links, previewRow{
			Label: formatHex(reference.To),
			Href:  functionPath(name, reference.To),
		})
	}
	return links
}

func noteBelongs(function decomp.Function, address uint64) bool {
	if function.Size == 0 {
		if function.Address == address {
			return true
		}
	} else if address >= function.Address && address < function.Address+function.Size {
		return true
	}
	for _, instruction := range function.Instructions {
		if instruction.Address == address {
			return true
		}
	}
	return false
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
