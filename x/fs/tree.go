package fs

import (
	"io"
	iofs "io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

// FS is a read-only tree of [File] members.
type FS struct {
	root *dnode
}

var (
	_ iofs.FS         = (*FS)(nil)
	_ iofs.ReadDirFS  = (*FS)(nil)
	_ iofs.ReadFileFS = (*FS)(nil)
	_ iofs.StatFS     = (*FS)(nil)
)

// New indexes files into a read-only filesystem.
// Implicit parent directories are created.
// A later regular file replaces an earlier one at the same name.
func New(files Files) (*FS, error) {
	root := newDir(".")
	for f, err := range files {
		if err != nil {
			return nil, err
		}
		if f.Name.String() == "." || f.Name.String() == "" {
			continue
		}
		if !f.Name.Valid() {
			return nil, &iofs.PathError{Op: "open", Path: f.Name.String(), Err: iofs.ErrInvalid}
		}
		if err := root.add(f); err != nil {
			return nil, err
		}
	}
	return &FS{root: root}, nil
}

// Open implements [io/fs.FS].
func (d *FS) Open(name string) (iofs.File, error) {
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return n.open()
}

// ReadDir implements [io/fs.ReadDirFS].
func (d *FS) ReadDir(name string) ([]iofs.DirEntry, error) {
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	if !n.file.Mode.IsDir() {
		return nil, &iofs.PathError{Op: "readdir", Path: name, Err: iofs.ErrInvalid}
	}
	return n.readDir(), nil
}

// ReadFile implements [io/fs.ReadFileFS].
func (d *FS) ReadFile(name string) ([]byte, error) {
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	f, err := n.file.Open()
	if err != nil {
		if n.file.Mode.IsDir() {
			return nil, &iofs.PathError{Op: "read", Path: name, Err: iofs.ErrInvalid}
		}
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Stat implements [io/fs.StatFS].
func (d *FS) Stat(name string) (iofs.FileInfo, error) {
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return n.file.info(), nil
}

func (d *FS) lookup(name string) (*dnode, error) {
	if name == "." || name == "" {
		return d.root, nil
	}
	if !iofs.ValidPath(name) {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
	}
	n := d.root.lookup(name)
	if n == nil {
		return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrNotExist}
	}
	return n, nil
}

type dnode struct {
	file File
	kids map[string]*dnode
}

func newDir(name string) *dnode {
	return &dnode{
		file: File{Name: path.New(name), Mode: iofs.ModeDir | 0o555},
		kids: map[string]*dnode{},
	}
}

func (n *dnode) add(f File) error {
	parts := f.Name.Parts()
	cur := n
	for i, p := range parts {
		if p == "" || p == "." || p == ".." || p == "/" {
			return &iofs.PathError{Op: "open", Path: f.Name.String(), Err: iofs.ErrInvalid}
		}
		next := cur.kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if f.Mode.IsDir() && next.file.Mode.IsDir() {
					if f.ModTime.After(next.file.ModTime) {
						next.file.ModTime = f.ModTime
					}
					return nil
				}
				if !f.Mode.IsDir() && !next.file.Mode.IsDir() {
					next.file = f
					return nil
				}
				return &iofs.PathError{Op: "open", Path: f.Name.String(), Err: iofs.ErrExist}
			}
			node := &dnode{file: f}
			if f.Mode.IsDir() {
				node.kids = map[string]*dnode{}
			}
			cur.kids[p] = node
			return nil
		}
		if next == nil {
			next = newDir(p)
			cur.kids[p] = next
		}
		if !next.file.Mode.IsDir() {
			return &iofs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: iofs.ErrInvalid}
		}
		cur = next
	}
	return nil
}

func (n *dnode) lookup(rel string) *dnode {
	cur := n
	for p := range strings.SplitSeq(rel, "/") {
		if cur == nil || !cur.file.Mode.IsDir() {
			return nil
		}
		cur = cur.kids[p]
	}
	return cur
}

func (n *dnode) open() (iofs.File, error) {
	if n.file.Mode.IsDir() {
		return &dirFile{n: n, infos: n.readDir()}, nil
	}
	return n.file.Open()
}
