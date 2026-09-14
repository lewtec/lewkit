package zip

import (
	stdzip "archive/zip"
	"io"
	"io/fs"
)

func openStored(ra io.ReaderAt, zf *stdzip.File) (fs.File, error) {
	off, err := zf.DataOffset()
	if err != nil {
		return nil, err
	}
	return &storedFile{
		info: zf.FileInfo(),
		r:    io.NewSectionReader(ra, off, int64(zf.UncompressedSize64)),
	}, nil
}

type storedFile struct {
	info fs.FileInfo
	r    *io.SectionReader
}

var (
	_ fs.File     = (*storedFile)(nil)
	_ io.ReaderAt = (*storedFile)(nil)
	_ io.Seeker   = (*storedFile)(nil)
)

func (f *storedFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *storedFile) Read(p []byte) (int, error) { return f.r.Read(p) }

func (f *storedFile) ReadAt(p []byte, off int64) (int, error) {
	return f.r.ReadAt(p, off)
}

func (f *storedFile) Seek(offset int64, whence int) (int64, error) {
	return f.r.Seek(offset, whence)
}

func (f *storedFile) Close() error { return nil }
