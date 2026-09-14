//go:build linux || windows

package wim

import (
	"io/fs"
	"strings"
	"time"

	winwim "github.com/Microsoft/go-winio/wim"
)

type dnode struct {
	name string
	dir  bool
	wf   *winwim.File
	kids map[string]*dnode
}

func newDir(name string) *dnode {
	return &dnode{name: name, dir: true, kids: map[string]*dnode{}}
}

func (n *dnode) add(rel string, wf *winwim.File, dir bool) error {
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
					if wf != nil {
						next.wf = wf
					}
					return nil
				}
				return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrExist}
			}
			node := &dnode{name: p, dir: dir, wf: wf}
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
	if n.wf != nil {
		size = n.wf.Size
		mod = n.wf.LastWriteTime.Time()
	}
	return fileInfo{name: n.name, size: size, mode: mode, mod: mod}
}
