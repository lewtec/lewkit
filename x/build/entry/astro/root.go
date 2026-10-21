// Package astro is the astro: entry provider.
// Importing it registers the provider. x/build/entry/prelude imports it.
package astro

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/build/entry"
	"github.com/lewtec/lewkit/x/build/gen/common"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/nodescript"
	"github.com/lucasew/orvalho/pkg/workers/bundle"
)

// Provider builds an Astro project and serves the bundled worker.
type Provider struct {
	*entry.Materializer
}

func init() {
	entry.Register(&Provider{Materializer: &entry.Materializer{}})
}

func (Provider) Name() string { return "astro" }

func (p *Provider) Main(ctx context.Context, request entry.Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	project, err := p.Resolve(request.Ref, request.Base)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(project)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", entry.ErrAstroConfig, project)
		}
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", entry.ErrAstroConfig, project)
	}
	print, err := p.fingerprint(project, info)
	if err != nil {
		return "", err
	}
	key := entry.CacheKey(p.Name(), project, request.Title, print)
	return p.Open(ctx, key, func(dir string) error {
		return p.write(ctx, project, dir, request.Title)
	})
}

func (p *Provider) write(ctx context.Context, project, dir, title string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	if err := p.requireConfig(project); err != nil {
		return err
	}
	if err := p.build(ctx, project); err != nil {
		return err
	}
	worker, err := os.MkdirTemp("", "lewkit-astro-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(worker)
	if err := p.assemble(project, worker); err != nil {
		return err
	}
	esbuild, err := p.findEsbuild(ctx, project)
	if err != nil {
		return err
	}
	script, err := bundle.BundleEntry(bundle.BundleOptions{
		PackageDir: worker,
		Entry:      "entry.mjs",
		Esbuild:    esbuild,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", entry.ErrAstroBuild, err)
	}
	embedDir := filepath.Join(dir, "embed")
	if err := os.MkdirAll(filepath.Join(embedDir, "assets"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(embedDir, "guest.js"), []byte(script), 0o644); err != nil {
		return err
	}
	if err := p.copyAssets(worker, embedDir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(p.source(title)), 0o644)
}

func (p *Provider) requireConfig(project string) error {
	for _, name := range []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"} {
		if fileExists(filepath.Join(project, name)) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", entry.ErrAstroConfig, project)
}

func (p *Provider) build(ctx context.Context, project string) error {
	if err := nodescript.Run(ctx, project, "build"); err != nil {
		return fmt.Errorf("%w: %w", entry.ErrAstroBuild, err)
	}
	return nil
}

func (p *Provider) assemble(project, worker string) error {
	if err := os.MkdirAll(filepath.Join(worker, "assets"), 0o755); err != nil {
		return err
	}
	server := filepath.Join(project, "dist", "server")
	client := filepath.Join(project, "dist", "client")
	legacy := filepath.Join(project, "dist", "_worker.js")
	switch {
	case dirExists(server):
		if err := common.CopyDir(server, worker); err != nil {
			return err
		}
		if dirExists(client) {
			if err := common.CopyDir(client, filepath.Join(worker, "assets")); err != nil {
				return err
			}
		}
	case dirExists(legacy):
		if err := common.CopyDir(legacy, worker); err != nil {
			return err
		}
		if err := p.copyLegacy(filepath.Join(project, "dist"), filepath.Join(worker, "assets")); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", entry.ErrAstroDist, filepath.Join(project, "dist"))
	}
	entryFile := filepath.Join(worker, "entry.mjs")
	indexFile := filepath.Join(worker, "index.js")
	if !fileExists(entryFile) && fileExists(indexFile) {
		if err := common.CopyFile(indexFile, entryFile, 0o644); err != nil {
			return err
		}
	}
	if !fileExists(entryFile) {
		return fmt.Errorf("%w: %s", entry.ErrAstroDist, entryFile)
	}
	return nil
}

func (p *Provider) copyLegacy(dist, assets string) error {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return err
	}
	for _, item := range entries {
		switch item.Name() {
		case "_worker.js", "_routes.json", "wrangler.json", "wrangler.toml":
			continue
		}
		src := filepath.Join(dist, item.Name())
		dst := filepath.Join(assets, item.Name())
		if item.IsDir() {
			if err := common.CopyDir(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := common.CopyFile(src, dst, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) copyAssets(worker, embedDir string) error {
	src := filepath.Join(worker, "assets")
	dst := filepath.Join(embedDir, "assets")
	if dirExists(src) {
		if err := common.CopyDir(src, dst); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.WriteFile(filepath.Join(dst, ".gitkeep"), nil, 0o644)
	}
	return nil
}

func (p *Provider) findEsbuild(ctx context.Context, project string) (string, error) {
	candidates := []string{
		filepath.Join(project, "node_modules", ".bin", "esbuild"),
		filepath.Join(project, "node_modules", "esbuild", "bin", "esbuild"),
	}
	if path := p.orvalhoEsbuild(project); path != "" {
		candidates = append(candidates, path)
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	path, err := execdriver.Which(ctx, "esbuild")
	if err != nil {
		return "", fmt.Errorf("%w: %w", entry.ErrEsbuild, err)
	}
	return path, nil
}

func (p *Provider) orvalhoEsbuild(project string) string {
	root := filepath.Join(project, "node_modules", ".orvalho")
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, item := range entries {
		if item.IsDir() && strings.HasPrefix(item.Name(), "esbuild@") {
			names = append(names, item.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	dir := filepath.Join(root, names[len(names)-1])
	for _, rel := range []string{"bin/esbuild", "esbuild"} {
		path := filepath.Join(dir, rel)
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func (p *Provider) fingerprint(dir string, info os.FileInfo) (string, error) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "dir %d\n", info.ModTime().UnixNano())
	names := []string{
		"package.json",
		"astro.config.mjs",
		"astro.config.ts",
		"astro.config.js",
		"bun.lock",
		"bun.lockb",
		"pnpm-lock.yaml",
		"package-lock.json",
	}
	for _, name := range names {
		fileInfo, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", err
		}
		fmt.Fprintf(&buf, "%s %d %d\n", name, fileInfo.Size(), fileInfo.ModTime().UnixNano())
	}
	return buf.String(), nil
}

func (Provider) source(title string) string {
	title = entry.DisplayTitle(title)
	return fmt.Sprintf(`package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lucasew/orvalho/pkg/workers"
)

//go:embed embed/guest.js
var guestJS string

//go:embed all:embed/assets
var assetsRoot embed.FS

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	assets, err := fs.Sub(assetsRoot, "embed/assets")
	if err != nil {
		return err
	}
	iso := workers.New(guestJS, workers.Options{
		Env: map[string]string{"SITE": %q},
		Bindings: map[string]workers.Binding{
			"ASSETS": workers.NewAssetBinding(assets, "."),
		},
		Fetch: workers.HTTPFetch(workers.EgressList{"*"}, nil, 0),
	})
	handler := workers.Handler(iso)
	return app.App{
		Title: %q,
		Handler: app.Web(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			recorder := &statusRecorder{ResponseWriter: response, code: http.StatusOK}
			handler.ServeHTTP(recorder, request)
			if recorder.code >= 500 {
				slog.Error("astro", "method", request.Method, "uri", request.URL.RequestURI(), "status", recorder.code)
			}
		})),
	}.Run(ctx)
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (recorder *statusRecorder) WriteHeader(code int) {
	recorder.code = code
	recorder.ResponseWriter.WriteHeader(code)
}
`, title, title)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
