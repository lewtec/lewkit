package tar

import (
	stdtar "archive/tar"
	"bytes"
	"io"
	"io/fs"
	stdpath "path"
	"strings"

	"github.com/lewtec/lewkit/x/compression"
	_ "github.com/lewtec/lewkit/x/compression/prelude"
	lewfs "github.com/lewtec/lewkit/x/fs"
)

// FS is a read-only tar archive.
type FS struct {
	ra   io.ReaderAt
	root *dnode
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Open reads a tar archive from r.
//
// A compressed wrapper is chosen by file name (if r has Stat) or by
// magic prefix. Compressed streams are decompressed into memory, then
// opened as an uncompressed tar. An uncompressed tar still needs
// [io.ReaderAt].
func Open(r io.Reader) (*FS, error) {
	name := nameOf(r)
	if ra, ok := r.(io.ReaderAt); ok {
		hdr := peekAt(ra, 16)
		if c, ok := compression.Detect(name, hdr); ok {
			return openCompressed(c, io.NewSectionReader(ra, 0, 1<<63-1))
		}
		return openTar(r)
	}
	hdr, br, err := peekCopy(r, 16)
	if err != nil {
		return nil, err
	}
	if c, ok := compression.Detect(name, hdr); ok {
		return openCompressed(c, br)
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: lewfs.ErrNeedReadAt}
}

func openCompressed(c compression.Codec, r io.Reader) (*FS, error) {
	cr, err := c.Reader(r)
	if err != nil {
		return nil, err
	}
	defer cr.Close()
	raw, err := io.ReadAll(cr)
	if err != nil {
		return nil, err
	}
	return openTar(bytes.NewReader(raw))
}

func openTar(r io.Reader) (*FS, error) {
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	cr := &cursor{ra: ra}
	tr := stdtar.NewReader(cr)
	root := newDir(".")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return &FS{ra: ra, root: root}, nil
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == stdtar.TypeXGlobalHeader {
			continue
		}
		name, err := cleanName(hdr.Name)
		if err != nil {
			return nil, err
		}
		if name == "." {
			continue
		}
		switch hdr.Typeflag {
		case stdtar.TypeDir:
			if err := root.add(name, meta{dir: true, mode: hdr.FileInfo().Mode(), mod: hdr.ModTime}); err != nil {
				return nil, err
			}
		case stdtar.TypeReg:
			if err := root.add(name, meta{
				off:  cr.off,
				size: hdr.Size,
				mode: hdr.FileInfo().Mode(),
				mod:  hdr.ModTime,
			}); err != nil {
				return nil, err
			}
		}
	}
}

func nameOf(r io.Reader) string {
	st, ok := r.(interface {
		Stat() (fs.FileInfo, error)
	})
	if !ok {
		return ""
	}
	fi, err := st.Stat()
	if err != nil {
		return ""
	}
	return fi.Name()
}

func peekAt(ra io.ReaderAt, n int) []byte {
	b := make([]byte, n)
	got, err := ra.ReadAt(b, 0)
	if got == 0 && err != nil {
		return nil
	}
	return b[:got]
}

func peekCopy(r io.Reader, n int) ([]byte, io.Reader, error) {
	b := make([]byte, n)
	got, err := io.ReadFull(r, b)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return nil, nil, err
	}
	hdr := b[:got]
	return hdr, io.MultiReader(bytes.NewReader(hdr), r), nil
}

func cleanName(name string) (string, error) {
	name = stdpath.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		name = "."
	}
	if name != "." && !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return name, nil
}

type cursor struct {
	ra  io.ReaderAt
	off int64
}

func (c *cursor) Read(p []byte) (int, error) {
	n, err := c.ra.ReadAt(p, c.off)
	c.off += int64(n)
	return n, err
}

func (c *cursor) Seek(offset int64, whence int) (int64, error) {
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = c.off + offset
	default:
		return 0, fs.ErrInvalid
	}
	if next < 0 {
		return 0, fs.ErrInvalid
	}
	c.off = next
	return next, nil
}
