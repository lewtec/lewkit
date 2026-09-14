package path

import (
	"io/fs"
	"iter"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// MatchGlob reports whether p matches pattern using doublestar.
func (p Path) MatchGlob(pattern string) (bool, error) {
	return doublestar.Match(pattern, p.s)
}

// Select yields names from seq that match pattern under p.
// Patterns use [github.com/bmatcuk/doublestar/v4], same as [Path.Glob].
func (p Path) Select(seq iter.Seq2[Path, error], pattern string) iter.Seq2[Path, error] {
	pat := globPattern(p, pattern)
	return func(yield func(Path, error) bool) {
		for name, err := range seq {
			if err != nil {
				yield(Path{}, err)
				return
			}
			ok, err := name.MatchGlob(pat)
			if err != nil {
				yield(Path{}, &fs.PathError{Op: "select", Path: pat, Err: err})
				return
			}
			if !ok {
				continue
			}
			if !yield(name, nil) {
				return
			}
		}
	}
}

// Under yields names from seq that are p or a child of p.
func (p Path) Under(seq iter.Seq2[Path, error]) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		for name, err := range seq {
			if err != nil {
				yield(Path{}, err)
				return
			}
			if !under(p.s, name.s) {
				continue
			}
			if !yield(name, nil) {
				return
			}
		}
	}
}

func under(root, name string) bool {
	if root == "." || root == "" {
		return true
	}
	if name == root {
		return true
	}
	return strings.HasPrefix(name, root+"/")
}
