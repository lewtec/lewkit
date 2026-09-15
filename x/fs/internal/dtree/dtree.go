// Package dtree is a path tree for read-only container adapters.
package dtree

import (
	"cmp"
	"io"
	"io/fs"
	"slices"
	"strings"
	"time"
)

// Node is one directory or file in a container listing.
type Node[T any] struct {
	Name string
	Dir  bool
	Val  T
	Kids map[string]*Node[T]
}

// NewDir returns an empty directory node.
func NewDir[T any](name string) *Node[T] {
	return &Node[T]{Name: name, Dir: true, Kids: map[string]*Node[T]{}}
}

// Add inserts rel under n. A second directory at the same name keeps
// the first and records val when it is non-zero. A file/dir clash or
// a second file is [fs.ErrExist].
func (n *Node[T]) Add(rel string, val T, dir bool) error {
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")
	cur := n
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrInvalid}
		}
		next := cur.Kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if dir && next.Dir {
					var zero T
					if any(val) != any(zero) {
						next.Val = val
					}
					return nil
				}
				return &fs.PathError{Op: "open", Path: rel, Err: fs.ErrExist}
			}
			node := &Node[T]{Name: p, Dir: dir, Val: val}
			if dir {
				node.Kids = map[string]*Node[T]{}
			}
			cur.Kids[p] = node
			return nil
		}
		if next == nil {
			next = NewDir[T](p)
			cur.Kids[p] = next
		}
		if !next.Dir {
			return &fs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: fs.ErrInvalid}
		}
		cur = next
	}
	return nil
}

// Lookup walks rel from n. Missing or a file in the middle is nil.
func (n *Node[T]) Lookup(rel string) *Node[T] {
	cur := n
	for p := range strings.SplitSeq(rel, "/") {
		if cur == nil || !cur.Dir {
			return nil
		}
		cur = cur.Kids[p]
	}
	return cur
}

// LookupPath is [Lookup] for an [io/fs] name. "." is n itself.
func LookupPath[T any](root *Node[T], name string) (*Node[T], error) {
	if name == "." || name == "" {
		return root, nil
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	n := root.Lookup(name)
	if n == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return n, nil
}

// Entries lists children of n, sorted by name.
func (n *Node[T]) Entries(info func(*Node[T]) fs.FileInfo) []fs.DirEntry {
	out := make([]fs.DirEntry, 0, len(n.Kids))
	for _, k := range n.Kids {
		out = append(out, fs.FileInfoToDirEntry(info(k)))
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

// Info is a [fs.FileInfo] for one node.
func Info(name string, size int64, mode fs.FileMode, mod time.Time) fs.FileInfo {
	return fileInfo{name: name, size: size, mode: mode, mod: mod}
}

type fileInfo struct {
	name string
	size int64
	mode fs.FileMode
	mod  time.Time
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return i.mod }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return nil }

// DirFile is a directory [fs.File] over a cached listing.
type DirFile struct {
	Info fs.FileInfo
	Ents []fs.DirEntry
	off  int
}

func (d *DirFile) Stat() (fs.FileInfo, error) { return d.Info, nil }

func (d *DirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.Info.Name(), Err: fs.ErrInvalid}
}

func (d *DirFile) Close() error { return nil }

func (d *DirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.off >= len(d.Ents) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := d.Ents[d.off:]
		d.off = len(d.Ents)
		return out, nil
	}
	end := d.off + n
	if end > len(d.Ents) {
		end = len(d.Ents)
	}
	out := d.Ents[d.off:end]
	d.off = end
	return out, nil
}
