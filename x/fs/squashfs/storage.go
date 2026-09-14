package squashfs

import (
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/diskfs/go-diskfs/backend"
)

type storage struct {
	ra  io.ReaderAt
	n   int64
	pos int64
}

var _ backend.Storage = (*storage)(nil)

func (s *storage) Read(p []byte) (int, error) {
	if s.pos >= s.n {
		return 0, io.EOF
	}
	n, err := s.ra.ReadAt(p, s.pos)
	s.pos += int64(n)
	return n, err
}

func (s *storage) ReadAt(p []byte, off int64) (int, error) {
	return s.ra.ReadAt(p, off)
}

func (s *storage) Seek(offset int64, whence int) (int64, error) {
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = s.pos + offset
	case io.SeekEnd:
		next = s.n + offset
	default:
		return 0, fs.ErrInvalid
	}
	if next < 0 {
		return 0, fs.ErrInvalid
	}
	s.pos = next
	return next, nil
}

func (s *storage) Stat() (fs.FileInfo, error) { return sizeInfo{n: s.n}, nil }

func (s *storage) Close() error { return nil }

func (s *storage) Sys() (*os.File, error) { return nil, backend.ErrNotSuitable }

func (s *storage) Writable() (backend.WritableFile, error) {
	return nil, backend.ErrIncorrectOpenMode
}

func (s *storage) Path() string { return "" }

type sizeInfo struct{ n int64 }

func (sizeInfo) Name() string       { return "" }
func (i sizeInfo) Size() int64      { return i.n }
func (sizeInfo) Mode() fs.FileMode  { return 0 }
func (sizeInfo) ModTime() time.Time { return time.Time{} }
func (sizeInfo) IsDir() bool        { return false }
func (sizeInfo) Sys() any           { return nil }
