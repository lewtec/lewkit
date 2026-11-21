package squashfs

import (
	"io/fs"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

type file struct {
	fs.File
}

func (f *file) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return f.File.Read(p)
}

type dirFile struct {
	info fs.FileInfo
	ents []fs.DirEntry
	off  int
}

func (d *dirFile) Stat() (fs.FileInfo, error) { return d.info, nil }

func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.Name(), Err: fs.ErrInvalid}
}

func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	return lewfs.DirEntries(d.ents, &d.off, n)
}
