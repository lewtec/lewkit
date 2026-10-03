package fs

import (
	"cmp"
	"context"
	"io"
	iofs "io/fs"
	"slices"
	"strings"
	"time"
)

// Index decorates a flat listing with directory lookup.
// Tar has no directory table, and a compose destination is a map of
// files, so both keep an Index. A filesystem that already walks its
// own directories does not build one.
//
// A later regular file replaces an earlier one at the same name.
// A later directory keeps the newer [Node.ModTime].
// A file and a directory at the same name is [io/fs.ErrExist].
type Index[T any] struct {
	root *Node[T]
	info func(*Node[T]) iofs.FileInfo
	open func(*Node[T]) (iofs.File, error)
}

// Node is one indexed path. A directory may keep a nil payload.
type Node[T any] struct {
	name string
	path string
	dir  bool
	mod  time.Time
	pay  *T
	kids map[string]*Node[T]
}

// FS is an [Index] of [File]. [New] builds it from a listing.
type FS = Index[File]

var (
	_ iofs.FS         = (*FS)(nil)
	_ iofs.ReadDirFS  = (*FS)(nil)
	_ iofs.ReadFileFS = (*FS)(nil)
	_ iofs.StatFS     = (*FS)(nil)
)

// NewIndex returns an empty index. info describes a node.
// open reads a file; directories are opened by Index.
func NewIndex[T any](info func(*Node[T]) iofs.FileInfo, open func(*Node[T]) (iofs.File, error)) *Index[T] {
	return &Index[T]{
		root: &Node[T]{name: ".", path: ".", dir: true, kids: map[string]*Node[T]{}},
		info: info,
		open: open,
	}
}

// New indexes files into a read-only filesystem.
// Implicit parent directories are created.
// A later regular file replaces an earlier one at the same name.
func New(ctx context.Context, files Files) (*FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	idx := NewIndex(fileNodeInfo, fileNodeOpen)
	for f, err := range files {
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, context.Cause(ctx)
		}
		name := f.Name.String()
		if name == "." || name == "" {
			continue
		}
		if !f.Name.Valid() {
			return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
		}
		stored := f
		if err := idx.Add(name, &stored, f.Mode.IsDir(), f.ModTime); err != nil {
			return nil, err
		}
	}
	return idx, nil
}

// Add inserts rel. One leading slash is trimmed.
// A segment "", ".", or ".." is [io/fs.ErrInvalid].
// A non-directory before the leaf is [io/fs.ErrInvalid] at that prefix.
// An existing directory is kept. A nil payload does not replace a stored one.
// An existing file is replaced. A file and a directory clash is [io/fs.ErrExist].
func (idx *Index[T]) Add(rel string, pay *T, dir bool, mod time.Time) error {
	if idx == nil || idx.root == nil {
		return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrInvalid}
	}
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		if dir && mod.After(idx.root.mod) {
			idx.root.mod = mod
		}
		if dir && idx.root.pay == nil && pay != nil {
			idx.root.pay = pay
		}
		return nil
	}
	parts := strings.Split(rel, "/")
	cur := idx.root
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrInvalid}
		}
		next := cur.kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if dir && next.dir {
					if mod.After(next.mod) {
						next.mod = mod
					}
					if next.pay == nil && pay != nil {
						next.pay = pay
					}
					return nil
				}
				if !dir && !next.dir {
					next.pay = pay
					next.mod = mod
					return nil
				}
				return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrExist}
			}
			node := &Node[T]{name: p, path: rel, dir: dir, mod: mod, pay: pay}
			if dir {
				node.kids = map[string]*Node[T]{}
			}
			cur.kids[p] = node
			return nil
		}
		if next == nil {
			prefix := strings.Join(parts[:i+1], "/")
			next = &Node[T]{name: p, path: prefix, dir: true, kids: map[string]*Node[T]{}}
			cur.kids[p] = next
		}
		if !next.dir {
			return &iofs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: iofs.ErrInvalid}
		}
		cur = next
	}
	return nil
}

// Name is the base name.
func (n *Node[T]) Name() string {
	if n == nil {
		return ""
	}
	return n.name
}

// Path is the slash path from the index root.
func (n *Node[T]) Path() string {
	if n == nil {
		return ""
	}
	return n.path
}

// IsDir reports whether n is a directory.
func (n *Node[T]) IsDir() bool { return n != nil && n.dir }

// ModTime is the node time. Directory adds keep the newer one.
func (n *Node[T]) ModTime() time.Time {
	if n == nil {
		return time.Time{}
	}
	return n.mod
}

// Payload is the stored record, or nil.
func (n *Node[T]) Payload() *T {
	if n == nil {
		return nil
	}
	return n.pay
}

// Info is an [io/fs.FileInfo] for this node.
func (n *Node[T]) Info(size int64, mode iofs.FileMode, mod time.Time) iofs.FileInfo {
	name := ""
	if n != nil {
		name = n.name
	}
	return FileInfo(name, size, mode, mod)
}

// Lookup resolves name from the root.
// "." and "" are the root. Any other name must pass [io/fs.ValidPath].
// A missing node is [io/fs.ErrNotExist].
func (idx *Index[T]) Lookup(name string) (*Node[T], error) {
	if idx == nil || idx.root == nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
	}
	if name == "." || name == "" {
		return idx.root, nil
	}
	if err := CheckName("open", name); err != nil {
		return nil, err
	}
	cur := idx.root
	for p := range strings.SplitSeq(name, "/") {
		if cur == nil || !cur.dir {
			return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
		}
		cur = cur.kids[p]
		if cur == nil {
			return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrNotExist}
		}
	}
	return cur, nil
}

// Open implements [io/fs.FS].
func (idx *Index[T]) Open(name string) (iofs.File, error) {
	n, err := idx.Lookup(name)
	if err != nil {
		return nil, err
	}
	if n.dir {
		return DirFile(name, idx.info(n), n.entries(idx.info)), nil
	}
	if idx.open == nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
	}
	return idx.open(n)
}

// ReadDir implements [io/fs.ReadDirFS].
func (idx *Index[T]) ReadDir(name string) ([]iofs.DirEntry, error) {
	n, err := idx.Lookup(name)
	if err != nil {
		return nil, err
	}
	if !n.dir {
		return nil, &iofs.PathError{Op: "readdir", Path: name, Err: iofs.ErrInvalid}
	}
	return n.entries(idx.info), nil
}

// ReadFile implements [io/fs.ReadFileFS].
func (idx *Index[T]) ReadFile(name string) ([]byte, error) {
	n, err := idx.Lookup(name)
	if err != nil {
		return nil, err
	}
	if n.dir {
		return nil, &iofs.PathError{Op: "read", Path: name, Err: iofs.ErrInvalid}
	}
	f, err := idx.open(n)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Stat implements [io/fs.StatFS].
func (idx *Index[T]) Stat(name string) (iofs.FileInfo, error) {
	n, err := idx.Lookup(name)
	if err != nil {
		return nil, err
	}
	return idx.info(n), nil
}

func (n *Node[T]) entries(info func(*Node[T]) iofs.FileInfo) []iofs.DirEntry {
	out := make([]iofs.DirEntry, 0, len(n.kids))
	for _, k := range n.kids {
		out = append(out, iofs.FileInfoToDirEntry(info(k)))
	}
	slices.SortFunc(out, func(a, b iofs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

func fileNodeInfo(n *Node[File]) iofs.FileInfo {
	if n.IsDir() {
		mode := iofs.ModeDir | 0o555
		if p := n.Payload(); p != nil && p.Mode != 0 {
			mode = p.Mode
			if !mode.IsDir() {
				mode |= iofs.ModeDir
			}
		}
		return n.Info(0, mode, n.ModTime())
	}
	if p := n.Payload(); p != nil {
		return p.info()
	}
	return n.Info(0, 0o444, n.ModTime())
}

func fileNodeOpen(n *Node[File]) (iofs.File, error) {
	p := n.Payload()
	if p == nil {
		return nil, &iofs.PathError{Op: "open", Path: n.Path(), Err: iofs.ErrInvalid}
	}
	return p.Open()
}
