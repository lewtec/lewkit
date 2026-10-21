// Package entry turns an eletrocromo entry into a Go main directory.
// go: is that directory. astro: and static: are a generated main
// under the lewkit checkout.
//
// The text is a tool spec: provider:ref. [tool.Parse] reads it.
// An empty entry is go:<go_main>, and an empty go_main is go:.
package entry

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool"
)

// Entry sentinels. Each one wraps the meta sentinel for its condition.
var (
	ErrEntryProvider = errs.Wrap(errs.ErrUnknown, "entry provider")
	ErrEntrySyntax   = errs.Wrap(errs.ErrSyntax, "entry")
	ErrStaticMissing = errs.Wrap(errs.ErrNotFound, "static site")
	ErrAstroConfig   = errs.Wrap(errs.ErrNotFound, "astro config")
	ErrAstroDist     = errs.Wrap(errs.ErrMissing, "astro dist")
	ErrAstroBuild    = errs.Wrap(errs.ErrInvalid, "astro build")
	ErrEsbuild       = errs.Wrap(errs.ErrNotFound, "esbuild")
	ErrLewkitSource  = errs.Wrap(errs.ErrNotFound, "lewkit module")
)

// Provider turns one entry reference into a Go main directory.
type Provider interface {
	Name() string
	Main(ctx context.Context, request Request) (string, error)
}

// Request is the reference a provider resolves.
// Ref is the text after provider:. Base resolves a relative ref.
// Title is the window title.
type Request struct {
	Ref   string
	Base  string
	Title string
}

// Entry is a parsed provider name and its reference.
type Entry struct {
	Name string
	Ref  string
}

var (
	providers     = map[string]Provider{}
	checkout      materializer
	staticSites   = &staticProvider{materializer: &checkout}
	astroProjects = &astroProvider{materializer: &checkout}
)

func init() {
	register(goProvider{})
	register(staticSites)
	register(astroProjects)
}

func register(provider Provider) {
	name := provider.Name()
	if _, exists := providers[name]; exists {
		panic("entry provider " + name + " already registered")
	}
	providers[name] = provider
}

func lookup(name string) (Provider, error) {
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrEntryProvider, name)
	}
	return provider, nil
}

// ParseEntry reads entry.
// An empty entry with an empty goMain is go:.
// An empty entry with a goMain is go:<goMain>.
// A reference is required. "." is a reference.
func ParseEntry(entry, goMain string) (Entry, error) {
	spec, err := parseSpec(entry, goMain)
	if err != nil {
		return Entry{}, err
	}
	if _, err := lookup(spec.Backend); err != nil {
		return Entry{}, err
	}
	return Entry{Name: spec.Backend, Ref: spec.Package}, nil
}

// Main returns the Go main directory for entry.
// The provider named by the tool spec does the work.
// ctx ends git reads and the Astro build.
func Main(ctx context.Context, entry, goMain, base, title string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	parsed, err := ParseEntry(entry, goMain)
	if err != nil {
		return "", err
	}
	provider, err := lookup(parsed.Name)
	if err != nil {
		return "", err
	}
	return provider.Main(ctx, Request{Ref: parsed.Ref, Base: base, Title: title})
}

func parseSpec(entry, goMain string) (tool.Spec, error) {
	entry = strings.TrimSpace(entry)
	goMain = strings.TrimSpace(goMain)
	if entry == "" {
		if goMain == "" {
			goMain = "."
		}
		return tool.Spec{Backend: "go", Package: goMain}, nil
	}
	if !strings.Contains(entry, ":") {
		return tool.Spec{}, fmt.Errorf("%w: %q", ErrEntrySyntax, entry)
	}
	spec, err := tool.Parse(entry)
	if err != nil {
		return tool.Spec{}, fmt.Errorf("%w: %w", ErrEntrySyntax, err)
	}
	spec.Backend = strings.TrimSpace(spec.Backend)
	spec.Package = strings.TrimSpace(spec.Package)
	if spec.Package == "" {
		return tool.Spec{}, fmt.Errorf("%w: %q", ErrEntrySyntax, entry)
	}
	return spec, nil
}

func displayTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "app"
	}
	return title
}

func cacheKey(name, ref, title, fingerprint string) string {
	return name + "\x00" + ref + "\x00" + displayTitle(title) + "\x00" + fingerprint
}

// materializer remembers a generated main and finds the lewkit checkout
// that main is compiled from.
type materializer struct {
	cache    sync.Map
	rootOnce sync.Once
	rootDir  string
	rootErr  error
}

func (m *materializer) open(ctx context.Context, key string, write func(dir string) error) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	if got, ok := m.cache.Load(key); ok {
		return got.(string), nil
	}
	root, err := m.lewkitRoot()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(root, ".lewkit-entry-")
	if err != nil {
		return "", err
	}
	if err := write(dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if existing, loaded := m.cache.LoadOrStore(key, dir); loaded {
		os.RemoveAll(dir)
		return existing.(string), nil
	}
	return dir, nil
}

func (m *materializer) resolve(ref, base string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w", ErrEntrySyntax)
	}
	if !filepath.IsAbs(ref) {
		ref = filepath.Join(base, ref)
	}
	return filepath.Abs(ref)
}

func (m *materializer) lewkitRoot() (string, error) {
	m.rootOnce.Do(func() {
		m.rootDir, m.rootErr = findLewkitRoot()
	})
	return m.rootDir, m.rootErr
}

func findLewkitRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", ErrLewkitSource
	}
	dir := filepath.Dir(file)
	for {
		body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && moduleLewkit(body) {
			if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr != nil {
				return "", fmt.Errorf("%w: %s", ErrLewkitSource, dir)
			}
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrLewkitSource
		}
		dir = parent
	}
}

func moduleLewkit(body []byte) bool {
	for _, line := range bytes.Split(body, []byte("\n")) {
		if bytes.Equal(bytes.TrimSpace(line), []byte("module github.com/lewtec/lewkit")) {
			return true
		}
	}
	return false
}
