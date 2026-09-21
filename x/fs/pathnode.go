package fs

import (
	"cmp"
	iofs "io/fs"
	"slices"
	"strings"
	"time"
)

// PathNode is a directory tree for archive adapters.
// [FS] is separate: it stores [File] and merges ModTime.
// T is the archive file record. A directory may keep a nil payload.
type PathNode[T any] struct {
	name string
	dir  bool
	pay  *T
	kids map[string]*PathNode[T]
}

// NewPathDir returns an empty directory named name.
func NewPathDir[T any](name string) *PathNode[T] {
	return &PathNode[T]{name: name, dir: true, kids: map[string]*PathNode[T]{}}
}

// Add inserts rel. One leading slash is trimmed.
// A segment "", ".", or ".." is [io/fs.ErrInvalid].
// A non-directory before the leaf is [io/fs.ErrInvalid] at that prefix.
// An existing directory leaf is kept. A non-nil payload replaces the stored one.
// Any other existing leaf is [io/fs.ErrExist].
func (n *PathNode[T]) Add(rel string, pay *T, dir bool) error {
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")
	cur := n
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrInvalid}
		}
		next := cur.kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if dir && next.dir {
					if pay != nil {
						next.pay = pay
					}
					return nil
				}
				return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrExist}
			}
			node := &PathNode[T]{name: p, dir: dir, pay: pay}
			if dir {
				node.kids = map[string]*PathNode[T]{}
			}
			cur.kids[p] = node
			return nil
		}
		if next == nil {
			next = NewPathDir[T](p)
			cur.kids[p] = next
		}
		if !next.dir {
			return &iofs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: iofs.ErrInvalid}
		}
		cur = next
	}
	return nil
}

func (n *PathNode[T]) walk(rel string) *PathNode[T] {
	cur := n
	for p := range strings.SplitSeq(rel, "/") {
		if cur == nil || !cur.dir {
			return nil
		}
		cur = cur.kids[p]
	}
	return cur
}

// Lookup resolves name from n.
// "." and "" are n. Any other name must pass [io/fs.ValidPath].
// A missing node is [io/fs.ErrNotExist].
func (n *PathNode[T]) Lookup(name string) (*PathNode[T], error) {
	if name == "." || name == "" {
		return n, nil
	}
	if !iofs.ValidPath(name) {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
	}
	hit := n.walk(name)
	if hit == nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrNotExist}
	}
	return hit, nil
}

// Name is the base name.
func (n *PathNode[T]) Name() string { return n.name }

// IsDir reports whether n is a directory.
func (n *PathNode[T]) IsDir() bool { return n.dir }

// Payload is the stored file record, or nil.
func (n *PathNode[T]) Payload() *T { return n.pay }

// Info is an [io/fs.FileInfo] for this node.
func (n *PathNode[T]) Info(size int64, mode iofs.FileMode, mod time.Time) iofs.FileInfo {
	return fileInfo{name: n.name, size: size, mode: mode, mod: mod}
}

// Entries lists children sorted by name. info builds each child's [io/fs.FileInfo].
func (n *PathNode[T]) Entries(info func(*PathNode[T]) iofs.FileInfo) []iofs.DirEntry {
	out := make([]iofs.DirEntry, 0, len(n.kids))
	for _, k := range n.kids {
		out = append(out, iofs.FileInfoToDirEntry(info(k)))
	}
	slices.SortFunc(out, func(a, b iofs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

// OpenDir returns a directory file.
// Stat uses info on this node. ReadDir pages the child list.
func (n *PathNode[T]) OpenDir(info func(*PathNode[T]) iofs.FileInfo) iofs.ReadDirFile {
	return &dirPage{
		name: n.name,
		stat: func() iofs.FileInfo { return info(n) },
		ents: n.Entries(info),
	}
}

type dirPage struct {
	name string
	stat func() iofs.FileInfo
	ents []iofs.DirEntry
	off  int
}

func (d *dirPage) Stat() (iofs.FileInfo, error) { return d.stat(), nil }

func (d *dirPage) Read([]byte) (int, error) {
	return 0, &iofs.PathError{Op: "read", Path: d.name, Err: iofs.ErrInvalid}
}

func (d *dirPage) Close() error { return nil }

func (d *dirPage) ReadDir(n int) ([]iofs.DirEntry, error) {
	return DirEntries(d.ents, &d.off, n)
}
