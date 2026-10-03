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

func (n udfNode) info() fs.FileInfo {
	if n.file == nil {
		return lewfs.FileInfo(".", 0, fs.ModeDir|0o555, time.Time{})
	}
	mode := fs.FileMode(0o444)
	if n.IsDir() {
		mode = fs.ModeDir | 0o555
	} else {
		mode = n.file.Mode() &^ fs.ModeDir
		if mode == 0 {
			mode = 0o444
		}
	}
	return lewfs.FileInfo(n.file.Name(), n.file.Size(), mode, n.file.ModTime())
}

func (n udfNode) open() (fs.File, error) {
	if n.file == nil || n.IsDir() {
		return nil, &fs.PathError{Op: "open", Path: n.Name(), Err: fs.ErrInvalid}
	}
	return &file{
		info: n.info(),
		r:    n.file.NewReader(),
	}, nil
}

func listEntries(n entry) ([]fs.DirEntry, error) {
	kids, err := n.children()
	if err != nil {
		return nil, err
	}
	out := make([]fs.DirEntry, 0, len(kids))
	for _, k := range kids {
		out = append(out, fs.FileInfoToDirEntry(k.info()))
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out, nil
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
