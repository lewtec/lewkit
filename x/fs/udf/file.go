package udf

import (
	"cmp"
	"io"
	"io/fs"
	"slices"
	"time"

	"github.com/Xmister/udf"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

func (n *dnode) open() (fs.File, error) {
	if n.dir {
		return &dirFile{n: n, infos: n.readDir()}, nil
	}
	if n.uf == nil {
		return nil, &fs.PathError{Op: "open", Path: n.name, Err: fs.ErrInvalid}
	}
	return &file{
		info: n.info(),
		r:    n.uf.NewReader(),
	}, nil
}

func (n *dnode) readDir() []fs.DirEntry {
	out := make([]fs.DirEntry, 0, len(n.kids))
	for _, k := range n.kids {
		out = append(out, fs.FileInfoToDirEntry(k.info()))
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

type file struct {
	info fileInfo
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

type dirFile struct {
	n     *dnode
	infos []fs.DirEntry
	off   int
}

func (d *dirFile) Stat() (fs.FileInfo, error) { return d.n.info(), nil }

func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.n.name, Err: fs.ErrInvalid}
}

func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	return lewfs.DirEntries(d.infos, &d.off, n)
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
