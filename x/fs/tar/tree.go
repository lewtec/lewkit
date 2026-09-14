package tar

import (
	"io/fs"
	"strings"
	"time"
)

type meta struct {
	dir  bool
	off  int64
	size int64
	mode fs.FileMode
	mod  time.Time
}

type dnode struct {
	name string
	dir  bool
	off  int64
	size int64
	mode fs.FileMode
	mod  time.Time
	kids map[string]*dnode
}

func newDir(name string) *dnode {
	return &dnode{name: name, dir: true, mode: fs.ModeDir | 0o555, kids: map[string]*dnode{}}
}

func (n *dnode) add(rel string, m meta) error {
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")
	cur := n
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrInvalid}
		}
		next := cur.kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if m.dir && next.dir {
					if m.mod.After(next.mod) {
						next.mod = m.mod
					}
					return nil
				}
				if !m.dir && !next.dir {
					next.off = m.off
					next.size = m.size
					next.mode = m.mode
					next.mod = m.mod
					return nil
				}
				return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrExist}
			}
			node := &dnode{name: p, dir: m.dir, off: m.off, size: m.size, mode: m.mode, mod: m.mod}
			if m.dir {
				if node.mode == 0 {
					node.mode = fs.ModeDir | 0o555
				}
				node.kids = map[string]*dnode{}
			}
			cur.kids[p] = node
			return nil
		}
		if next == nil {
			next = newDir(p)
			cur.kids[p] = next
		}
		if !next.dir {
			return &fs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: fs.ErrInvalid}
		}
		cur = next
	}
	return nil
}

func (n *dnode) lookup(rel string) *dnode {
	cur := n
	for p := range strings.SplitSeq(rel, "/") {
		if cur == nil || !cur.dir {
			return nil
		}
		cur = cur.kids[p]
	}
	return cur
}

func (n *dnode) info() fileInfo {
	mode := n.mode
	if mode == 0 {
		mode = 0o444
	}
	if n.dir {
		if mode&fs.ModeDir == 0 {
			mode |= fs.ModeDir | 0o555
		}
	}
	return fileInfo{name: n.name, size: n.size, mode: mode, mod: n.mod}
}
