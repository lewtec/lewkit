package entry

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

// staticProvider serves a zip or a directory from a generated main.
type staticProvider struct {
	*materializer
}

func (staticProvider) Name() string { return "static" }

func (p *staticProvider) Main(ctx context.Context, request Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	ref, err := p.resolve(request.Ref, request.Base)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(ref)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrStaticMissing, ref)
		}
		return "", err
	}
	print, err := p.fingerprint(ctx, ref, info)
	if err != nil {
		return "", err
	}
	key := cacheKey(p.Name(), ref, request.Title, print)
	return p.open(ctx, key, func(dir string) error {
		return p.write(ctx, ref, info, dir, request.Title)
	})
}

func (p *staticProvider) write(ctx context.Context, ref string, info os.FileInfo, dir, title string) error {
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

func (p *staticProvider) fingerprint(ctx context.Context, ref string, info os.FileInfo) (string, error) {
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

func (staticProvider) zipSource(title string) string {
	title = displayTitle(title)
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

func (staticProvider) dirSource(title string) string {
	title = displayTitle(title)
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
