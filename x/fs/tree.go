package fs

import (
	"io"
	iofs "io/fs"
	"strings"
	"time"
)

// FS is a read-only tree built from a [Files] listing.
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
		name := f.Name.String()
		if name == "." || name == "" {
			continue
		}
		if !f.Name.Valid() {
			return nil, &iofs.PathError{Op: "open", Path: name, Err: iofs.ErrInvalid}
		}
		if err := root.add(name, f); err != nil {
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
	if !n.dir {
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
	if n.dir {
		return nil, &iofs.PathError{Op: "read", Path: name, Err: iofs.ErrInvalid}
	}
	if n.openFn == nil {
		return nil, &iofs.PathError{Op: "read", Path: name, Err: iofs.ErrInvalid}
	}
	f, err := n.openFn()
	if err != nil {
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
	return n.info(), nil
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
	name   string
	dir    bool
	size   int64
	mode   iofs.FileMode
	mod    time.Time
	openFn func() (iofs.File, error)
	kids   map[string]*dnode
}

func newDir(name string) *dnode {
	return &dnode{name: name, dir: true, mode: iofs.ModeDir | 0o555, kids: map[string]*dnode{}}
}

func (n *dnode) add(rel string, f File) error {
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")
	dir := f.Mode.IsDir()
	cur := n
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrInvalid}
		}
		next := cur.kids[p]
		if i == len(parts)-1 {
			if next != nil {
				if dir && next.dir {
					if f.ModTime.After(next.mod) {
						next.mod = f.ModTime
					}
					return nil
				}
				if !dir && !next.dir {
					next.size = f.Size
					next.mode = f.Mode
					next.mod = f.ModTime
					next.openFn = f.Open
					return nil
				}
				return &iofs.PathError{Op: "open", Path: rel, Err: iofs.ErrExist}
			}
			node := &dnode{name: p, dir: dir, size: f.Size, mode: f.Mode, mod: f.ModTime, openFn: f.Open}
			if dir {
				if node.mode == 0 {
					node.mode = iofs.ModeDir | 0o555
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
			return &iofs.PathError{Op: "open", Path: strings.Join(parts[:i+1], "/"), Err: iofs.ErrInvalid}
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
		if mode&iofs.ModeDir == 0 {
			mode |= iofs.ModeDir | 0o555
		}
	}
	return fileInfo{name: n.name, size: n.size, mode: mode, mod: n.mod}
}

func (n *dnode) open() (iofs.File, error) {
	if n.dir {
		return &dirFile{n: n, infos: n.readDir()}, nil
	}
	if n.openFn == nil {
		return nil, &iofs.PathError{Op: "open", Path: n.name, Err: iofs.ErrInvalid}
	}
	return n.openFn()
}
