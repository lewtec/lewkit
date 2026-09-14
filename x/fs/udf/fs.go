package udf

import (
	"io"
	"io/fs"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (f fs.File, err error) {
	defer recovered(&err)
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return n.open()
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) (ents []fs.DirEntry, err error) {
	defer recovered(&err)
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
func (d *FS) ReadFile(name string) (b []byte, err error) {
	defer recovered(&err)
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	if n.dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	if n.uf == nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	r := n.uf.NewReader()
	buf := make([]byte, n.uf.Size())
	_, err = io.ReadFull(r, buf)
	return buf, err
}

// Stat implements [fs.StatFS].
func (d *FS) Stat(name string) (fi fs.FileInfo, err error) {
	defer recovered(&err)
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
