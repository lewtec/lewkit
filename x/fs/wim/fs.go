//go:build linux || windows

package wim

import (
	"io"
	"io/fs"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (fs.File, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrClosed}
	}
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return openNode(n)
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrClosed}
	}
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	if !n.Dir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	return n.Entries(nodeInfo), nil
}

// ReadFile implements [fs.ReadFileFS].
func (d *FS) ReadFile(name string) ([]byte, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrClosed}
	}
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	if n.Dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	f, err := openNode(n)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// Stat implements [fs.StatFS].
func (d *FS) Stat(name string) (fs.FileInfo, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrClosed}
	}
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return nodeInfo(n), nil
}

func (d *FS) lookup(name string) (*dnode, error) {
	return dtree.LookupPath(d.root, name)
}
