package tar

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	stdpath "path"
	"strings"

	"github.com/lewtec/lewkit/x/compression"
	_ "github.com/lewtec/lewkit/x/compression/prelude"
	lewfs "github.com/lewtec/lewkit/x/fs"
)

// Open reads a tar archive from r and indexes [Files] with [lewfs.New].
//
// A compressed wrapper is chosen by file name (if r has Stat) or by
// magic prefix, then decompressed into memory. An uncompressed tar
// still needs [io.ReaderAt].
func Open(ctx context.Context, r io.Reader) (*lewfs.FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	name := nameOf(r)
	r, err := uncompressed(ctx, r)
	if err != nil {
		return nil, err
	}
	if _, err := lewfs.ReaderAt("open", r); err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: lewfs.ErrNeedReadAt}
	}
	return lewfs.New(ctx, Files(ctx, r))
}

func uncompressed(ctx context.Context, r io.Reader) (io.Reader, error) {
	name := nameOf(r)
	if ra, ok := r.(io.ReaderAt); ok {
		if c, ok := compression.Detect(name, peekAt(ra, 16)); ok {
			return decompress(ctx, c, io.NewSectionReader(ra, 0, 1<<63-1))
		}
		return r, nil
	}
	hdr, src, err := peekCopy(r, 16)
	if err != nil {
		return nil, err
	}
	if c, ok := compression.Detect(name, hdr); ok {
		return decompress(ctx, c, src)
	}
	return src, nil
}

func decompress(ctx context.Context, c compression.Codec, r io.Reader) (io.Reader, error) {
	d, ok := c.(compression.Decompressor)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: "", Err: fs.ErrInvalid}
	}
	cr, err := d.Reader(r)
	if err != nil {
		return nil, err
	}
	defer cr.Close()
	raw, err := io.ReadAll(lewfs.ContextReader(ctx, cr))
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(raw), nil
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
	if err := lewfs.CheckName("open", name); err != nil {
		return "", err
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
