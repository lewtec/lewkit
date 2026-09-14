//go:build linux || windows

package wim

import (
	"io"
	"io/fs"
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
	return n.open()
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
	if !n.dir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	return n.readDir(), nil
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
	if n.dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	f, err := n.open()
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
	return n.info(), nil
}

func (d *FS) lookup(name string) (*dnode, error) {
	if name == "." || name == "" {
		return d.root, nil
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	n := d.root.lookup(name)
	if n == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return n, nil
}
