package path

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var errStopGlob = errors.New("stop glob")

// Glob yields names under p that match pattern.
// Patterns use [github.com/bmatcuk/doublestar/v4]: "**" is a whole segment
// and does not follow symlinks. A trailing slash keeps only directories.
func (p Path) Glob(ctx context.Context, fsys fs.FS, pattern string) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(Path{}, context.Cause(ctx))
			return
		}
		pat := globPattern(p, pattern)
		seen := make(map[string]struct{})
		err := doublestar.GlobWalk(fsys, pat, func(name string, _ fs.DirEntry) error {
			if err := ctx.Err(); err != nil {
				yield(Path{}, context.Cause(ctx))
				return errStopGlob
			}
			if _, ok := seen[name]; ok {
				return nil
			}
			seen[name] = struct{}{}
			if !yield(Path{s: name}, nil) {
				return errStopGlob
			}
			return nil
		}, doublestar.WithNoFollow())
		if err == nil || errors.Is(err, errStopGlob) {
			return
		}
		if errors.Is(err, doublestar.ErrBadPattern) {
			yield(Path{}, &fs.PathError{Op: "glob", Path: pat, Err: err})
			return
		}
		yield(Path{}, err)
	}
}

// Rglob is [Path.Glob] with "**/" in front of pattern.
func (p Path) Rglob(ctx context.Context, fsys fs.FS, pattern string) iter.Seq2[Path, error] {
	return p.Glob(ctx, fsys, "**/"+pattern)
}

func globPattern(p Path, pattern string) string {
	pat := p.Join(pattern).s
	if strings.HasSuffix(pattern, "/") && !strings.HasSuffix(pat, "/") {
		return pat + "/"
	}
	return pat
}
