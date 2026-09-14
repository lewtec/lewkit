package udf

import (
	"io/fs"
	"strings"
	"time"

	"github.com/Xmister/udf"
)

type dnode struct {
	name string
	dir  bool
	uf   *udf.File
	kids map[string]*dnode
}

func newDir(name string) *dnode {
	return &dnode{name: name, dir: true, kids: map[string]*dnode{}}
}

func (n *dnode) add(rel string, uf *udf.File, dir bool) error {
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
				if dir && next.dir {
					if uf != nil {
						next.uf = uf
					}
					return nil
				}
				return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrExist}
			}
			node := &dnode{name: p, dir: dir, uf: uf}
			if dir {
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
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	if n.dir {
		mode = fs.ModeDir | 0o555
	}
	if n.uf != nil {
		size = n.uf.Size()
		mod = n.uf.ModTime()
		if !n.dir {
			mode = n.uf.Mode() &^ fs.ModeDir
			if mode == 0 {
				mode = 0o444
			}
		}
	}
	return fileInfo{name: n.name, size: size, mode: mode, mod: mod}
}
