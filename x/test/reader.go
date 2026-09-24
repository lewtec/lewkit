package test

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// OnlyReader is an io.Reader that does not implement io.ReaderAt.
type OnlyReader struct{ io.Reader }

// NeedReadAtOpen fails tb unless err is an open fs.PathError wrapping x/fs.ErrNeedReadAt.
func NeedReadAtOpen(tb testing.TB, err error) {
	tb.Helper()
	if !errors.Is(err, lewfs.ErrNeedReadAt) {
		tb.Fatalf("err = %v, want ErrNeedReadAt", err)
	}
	pe, ok := errors.AsType[*fs.PathError](err)
	if !ok {
		tb.Fatalf("err = %T, want *fs.PathError", err)
	}
	if pe.Op != "open" {
		tb.Fatalf("op = %q, want open", pe.Op)
	}
}

// ReaderOnlyFS returns an fs.FS that opens name as a reader-only file.
// The file has no ReadAt. Stat returns fs.ErrInvalid. Close returns nil.
// Any other name returns fs.ErrNotExist.
func ReaderOnlyFS(name string, r io.Reader) fs.FS {
	return &readerOnlyFS{name: name, r: r}
}

type readerOnlyFS struct {
	name string
	r    io.Reader
}

func (s *readerOnlyFS) Open(name string) (fs.File, error) {
	if name != s.name {
		return nil, fs.ErrNotExist
	}
	return readerOnlyFile{Reader: s.r}, nil
}

type readerOnlyFile struct{ io.Reader }

func (readerOnlyFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (readerOnlyFile) Close() error               { return nil }
