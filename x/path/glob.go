package path

import (
	"io/fs"
	"iter"
	stdpath "path"
	"strings"
)

// Glob yields names under p that match pattern.
// A "**" segment matches zero or more directories and does not follow
// symlinks. Other segments use [path.Match]. A trailing slash keeps
// only directories.
func (p Path) Glob(fsys fs.FS, pattern string) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		segs, onlyDirs, err := parseGlob(pattern)
		if err != nil {
			yield(Path{}, &fs.PathError{Op: "glob", Path: p.Join(pattern).s, Err: err})
			return
		}
		g := globber{
			fsys:     fsys,
			onlyDirs: onlyDirs,
			seen:     make(map[string]struct{}),
			yield:    yield,
		}
		g.walk(p, segs)
	}
}

// Rglob is [Path.Glob] with "**/" in front of pattern.
func (p Path) Rglob(fsys fs.FS, pattern string) iter.Seq2[Path, error] {
	return p.Glob(fsys, "**/"+pattern)
}

func parseGlob(pattern string) (segs []string, onlyDirs bool, err error) {
	if pattern == "" || stdpath.IsAbs(pattern) {
		return nil, false, stdpath.ErrBadPattern
	}
	onlyDirs = strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	if pattern == "" {
		return nil, false, stdpath.ErrBadPattern
	}
	segs = strings.Split(pattern, "/")
	for _, s := range segs {
		if s == "" || s == "." || s == ".." {
			return nil, false, stdpath.ErrBadPattern
		}
		if s == "**" {
			continue
		}
		if _, err := stdpath.Match(s, ""); err != nil {
			return nil, false, err
		}
	}
	return segs, onlyDirs, nil
}

type globber struct {
	fsys     fs.FS
	onlyDirs bool
	seen     map[string]struct{}
	yield    func(Path, error) bool
}

func (g *globber) walk(cur Path, segs []string) bool {
	if len(segs) == 0 {
		return g.emit(cur)
	}
	head, rest := segs[0], segs[1:]
	if head == "**" {
		if !g.walk(cur, rest) {
			return false
		}
		return g.star(cur, segs, rest)
	}
	return g.seg(cur, head, rest)
}

func (g *globber) emit(cur Path) bool {
	if g.onlyDirs {
		ok, err := cur.IsDir(g.fsys)
		if err != nil {
			return g.yield(Path{}, err)
		}
		if !ok {
			return true
		}
	}
	if _, ok := g.seen[cur.s]; ok {
		return true
	}
	g.seen[cur.s] = struct{}{}
	return g.yield(cur, nil)
}

func (g *globber) star(cur Path, segs, rest []string) bool {
	ents, err := globReadDir(g.fsys, cur)
	if err != nil {
		return g.yield(Path{}, err)
	}
	for _, e := range ents {
		child := cur.Join(e.Name())
		if e.IsDir() && e.Type()&fs.ModeSymlink == 0 {
			if !g.walk(child, segs) {
				return false
			}
			continue
		}
		if !g.leaf(child, e.Name(), rest) {
			return false
		}
	}
	return true
}

func (g *globber) leaf(cur Path, name string, rest []string) bool {
	switch len(rest) {
	case 0:
		return g.walk(cur, nil)
	case 1:
		ok, err := stdpath.Match(rest[0], name)
		if err != nil {
			return g.yield(Path{}, err)
		}
		if !ok {
			return true
		}
		return g.walk(cur, nil)
	default:
		return true
	}
}

func (g *globber) seg(cur Path, head string, rest []string) bool {
	ents, err := globReadDir(g.fsys, cur)
	if err != nil {
		return g.yield(Path{}, err)
	}
	for _, e := range ents {
		ok, err := stdpath.Match(head, e.Name())
		if err != nil {
			return g.yield(Path{}, err)
		}
		if !ok {
			continue
		}
		if !g.walk(cur.Join(e.Name()), rest) {
			return false
		}
	}
	return true
}

func globReadDir(fsys fs.FS, p Path) ([]fs.DirEntry, error) {
	st, err := fs.Stat(fsys, p.s)
	if err != nil {
		if missing(err) {
			return nil, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, nil
	}
	return fs.ReadDir(fsys, p.s)
}
