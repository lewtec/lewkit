package fs

import (
	iofs "io/fs"
	"time"
)

// FileInfo is an [io/fs.FileInfo] with the given fields.
func FileInfo(name string, size int64, mode iofs.FileMode, mod time.Time) iofs.FileInfo {
	return fileInfo{name: name, size: size, mode: mode, mod: mod}
}

// DirFile is a read-only directory. Stat returns info. ReadDir pages ents.
func DirFile(name string, info iofs.FileInfo, ents []iofs.DirEntry) iofs.ReadDirFile {
	return &dirPage{
		name: name,
		stat: func() iofs.FileInfo { return info },
		ents: ents,
	}
}

type dirPage struct {
	name string
	stat func() iofs.FileInfo
	ents []iofs.DirEntry
	off  int
}

func (d *dirPage) Stat() (iofs.FileInfo, error) { return d.stat(), nil }

func (d *dirPage) Read([]byte) (int, error) {
	return 0, &iofs.PathError{Op: "read", Path: d.name, Err: iofs.ErrInvalid}
}

func (d *dirPage) Close() error { return nil }

func (d *dirPage) ReadDir(n int) ([]iofs.DirEntry, error) {
	return DirEntries(d.ents, &d.off, n)
}

type fileInfo struct {
	name string
	size int64
	mode iofs.FileMode
	mod  time.Time
}

func (i fileInfo) Name() string        { return i.name }
func (i fileInfo) Size() int64         { return i.size }
func (i fileInfo) Mode() iofs.FileMode { return i.mode }
func (i fileInfo) ModTime() time.Time  { return i.mod }
func (i fileInfo) IsDir() bool         { return i.mode.IsDir() }
func (i fileInfo) Sys() any            { return nil }
