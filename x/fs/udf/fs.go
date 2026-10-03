package udf

import (
	"io"
	"io/fs"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (f fs.File, err error) {
	defer recovered(&err)
	n, err := d.lookup("open", name)
	if err != nil {
		return nil, err
	}
	if n.IsDir() {
		ents, err := listEntries(n)
		if err != nil {
			return nil, err
		}
		return lewfs.DirFile(name, n.info(), ents), nil
	}
	return n.open()
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) (ents []fs.DirEntry, err error) {
	defer recovered(&err)
	n, err := d.lookup("readdir", name)
	if err != nil {
		return nil, err
	}
	if !n.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	return listEntries(n)
}

// ReadFile implements [fs.ReadFileFS].
func (d *FS) ReadFile(name string) (b []byte, err error) {
	defer recovered(&err)
	n, err := d.lookup("read", name)
	if err != nil {
		return nil, err
	}
	if n.IsDir() {
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
func (d *FS) Stat(name string) (fi fs.FileInfo, err error) {
	defer recovered(&err)
	n, err := d.lookup("stat", name)
	if err != nil {
		return nil, err
	}
	return n.info(), nil
}

func (d *FS) lookup(op, name string) (entry, error) {
	if d == nil || d.root == nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	return lewfs.Lookup(op, name, d.root, func(e entry) ([]entry, error) {
		return e.children()
	})
}
