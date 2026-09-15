package udf

import (
	"io"
	"io/fs"

	"github.com/Xmister/udf"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

func openNode(n *dnode) (fs.File, error) {
	if n.Dir {
		return &dtree.DirFile{Info: nodeInfo(n), Ents: n.Entries(nodeInfo)}, nil
	}
	if n.Val == nil {
		return nil, &fs.PathError{Op: "open", Path: n.Name, Err: fs.ErrInvalid}
	}
	return &file{
		info: nodeInfo(n),
		r:    n.Val.NewReader(),
	}, nil
}

type file struct {
	info fs.FileInfo
	r    *udf.MultiSectionReader
}

var (
	_ fs.File     = (*file)(nil)
	_ io.ReaderAt = (*file)(nil)
	_ io.Seeker   = (*file)(nil)
)

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *file) Read(p []byte) (n int, err error) {
	defer recovered(&err)
	return f.r.Read(p)
}

func (f *file) ReadAt(p []byte, off int64) (n int, err error) {
	defer recovered(&err)
	return f.r.ReadAt(p, off)
}

func (f *file) Seek(offset int64, whence int) (n int64, err error) {
	defer recovered(&err)
	return f.r.Seek(offset, whence)
}

func (f *file) Close() error { return nil }
