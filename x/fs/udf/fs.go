package udf

import (
	"io"
	"io/fs"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (f fs.File, err error) {
	defer recovered(&err)
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	return openNode(n)
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) (ents []fs.DirEntry, err error) {
	defer recovered(&err)
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
func (d *FS) ReadFile(name string) (b []byte, err error) {
	defer recovered(&err)
	n, err := d.lookup(name)
	if err != nil {
		return nil, err
	}
	if n.Dir {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	if n.Val == nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	r := n.Val.NewReader()
	buf := make([]byte, n.Val.Size())
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
	return nodeInfo(n), nil
}

func (d *FS) lookup(name string) (*dnode, error) {
	return dtree.LookupPath(d.root, name)
}
