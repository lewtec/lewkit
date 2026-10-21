// Package entry turns an eletrocromo entry into a Go main directory.
// go: is that directory. astro: and static: are a generated main
// under the lewkit checkout.
package entry

import (
	"context"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/errs"
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

// Provider is the scheme of an eletrocromo entry.
//
//lewkit:enum
type Provider string

const (
	ProviderGo     Provider = "go"
	ProviderAstro  Provider = "astro"
	ProviderStatic Provider = "static"
)

// Entry is one provider and its reference.
// The reference is a Go main directory, an Astro project, or a site.
type Entry struct {
	Provider Provider
	Ref      string
}

// String is provider:ref.
func (e Entry) String() string {
	return string(e.Provider) + ":" + e.Ref
}

// ParseEntry reads entry.
// An empty entry with an empty goMain is go:.
// An empty entry with a goMain is go:<goMain>.
// A reference is required. "." is a reference.
func ParseEntry(entry, goMain string) (Entry, error) {
	entry = strings.TrimSpace(entry)
	goMain = strings.TrimSpace(goMain)
	if entry == "" {
		if goMain == "" {
			goMain = "."
		}
		return Entry{Provider: ProviderGo, Ref: goMain}, nil
	}
	name, ref, ok := strings.Cut(entry, ":")
	if !ok {
		return Entry{}, fmt.Errorf("%w: %q", ErrEntrySyntax, entry)
	}
	name = strings.TrimSpace(name)
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Entry{}, fmt.Errorf("%w: %q", ErrEntrySyntax, entry)
	}
	provider, err := ParseProvider(name)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %w", ErrEntryProvider, err)
	}
	return Entry{Provider: provider, Ref: ref}, nil
}

// Main returns the Go main directory for entry.
// go: resolves ref against base. astro: and static: write a main
// that serves that project. title is the window title.
// ctx ends git reads and the Astro build.
func Main(ctx context.Context, entry, goMain, base, title string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	parsed, err := ParseEntry(entry, goMain)
	if err != nil {
		return "", err
	}
	if parsed.Provider == ProviderGo {
		return common.ResolveGoMain(parsed.Ref, base)
	}
	return materialize(ctx, parsed, base, title)
}
