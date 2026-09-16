package zip

import (
	stdzip "archive/zip"
	"context"
	"io"
	"io/fs"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// FS is a read-only ZIP archive.
type FS struct {
	r  *stdzip.Reader
	ra io.ReaderAt
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Open reads a ZIP archive from r. r must be an [io.ReaderAt] with a known size.
func Open(ctx context.Context, r io.Reader) (*FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	size, err := lewfs.Size("open", r)
	if err != nil {
		return nil, err
	}
	zr, err := stdzip.NewReader(lewfs.ContextReaderAt(ctx, ra), size)
	if err != nil {
		return nil, err
	}
	return &FS{r: zr, ra: ra}, nil
}

func (d *FS) lookup(name string) *stdzip.File {
	for _, f := range d.r.File {
		n := strings.TrimSuffix(f.Name, "/")
		if n == name {
			return f
		}
	}
	return nil
}
