package tar

import (
	"io"
	"io/fs"
	"time"
)

type file struct {
	info fileInfo
	r    *io.SectionReader
}

var (
	_ fs.File     = (*file)(nil)
	_ io.ReaderAt = (*file)(nil)
	_ io.Seeker   = (*file)(nil)
)

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *file) Read(p []byte) (int, error) { return f.r.Read(p) }

func (f *file) ReadAt(p []byte, off int64) (int, error) { return f.r.ReadAt(p, off) }

func (f *file) Seek(offset int64, whence int) (int64, error) { return f.r.Seek(offset, whence) }

func (f *file) Close() error { return nil }

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
