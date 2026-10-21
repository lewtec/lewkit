// Package staticsite is the static: entry provider.
// Importing it registers the provider. x/build/entry/prelude imports it.
package staticsite

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/entry"
	"github.com/lewtec/lewkit/x/build/gen/common"
)

// Provider serves a zip or a directory from a generated main.
type Provider struct {
	*entry.Materializer
}

func init() {
	entry.Register(&Provider{Materializer: &entry.Materializer{}})
}

func (Provider) Name() string { return "static" }

func (p *Provider) Main(ctx context.Context, request entry.Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	ref, err := p.Resolve(request.Ref, request.Base)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(ref)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", entry.ErrStaticMissing, ref)
		}
		return "", err
	}
	print, err := p.fingerprint(ctx, ref, info)
	if err != nil {
		return "", err
	}
	key := entry.CacheKey(p.Name(), ref, request.Title, print)
	return p.Open(ctx, key, func(dir string) error {
		return p.write(ctx, ref, info, dir, request.Title)
	})
}

func (p *Provider) write(ctx context.Context, ref string, info os.FileInfo, dir, title string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	if info.IsDir() {
		if err := common.CopyDir(ref, filepath.Join(dir, "site")); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "main.go"), []byte(p.dirSource(title)), 0o644)
	}
	if err := common.CopyFile(ref, filepath.Join(dir, "site.zip"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(p.zipSource(title)), 0o644)
}

func (p *Provider) fingerprint(ctx context.Context, ref string, info os.FileInfo) (string, error) {
	if !info.IsDir() {
		return fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano()), nil
	}
	var buf strings.Builder
	err := filepath.WalkDir(ref, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.IsDir() {
			return nil
		}
		fileInfo, err := item.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(ref, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&buf, "%s %d %d\n", rel, fileInfo.Size(), fileInfo.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (Provider) zipSource(title string) string {
	title = entry.DisplayTitle(title)
	return fmt.Sprintf(`package main

import (
	"bytes"
	"context"
	_ "embed"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/site"
)

//go:embed site.zip
var siteZip []byte

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	handler, err := site.Zip(ctx, bytes.NewReader(siteZip))
	if err != nil {
		return err
	}
	return app.App{Title: %q, Handler: app.Web(handler)}.Run(ctx)
}
`, title)
}

func (Provider) dirSource(title string) string {
	title = entry.DisplayTitle(title)
	return fmt.Sprintf(`package main

import (
	"context"
	"embed"
	"io/fs"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/http/site"
)

//go:embed all:site
var siteRoot embed.FS

func init() { entry.Bind(run) }

func main() { entry.Main(context.Background(), run) }

func run(ctx context.Context) error {
	files, err := fs.Sub(siteRoot, "site")
	if err != nil {
		return err
	}
	return app.App{Title: %q, Handler: app.Web(site.Dir(files))}.Run(ctx)
}
`, title)
}
