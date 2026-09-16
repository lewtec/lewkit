package path

import (
	"context"
	"io"
	"io/fs"
)

// OpenFS opens p on fsys and passes the file to open.
// If open fails, the file is closed. On success the file stays open
// for the returned value (it must be an [io.ReaderAt] when the
// adapter needs random access).
func OpenFS[T any](ctx context.Context, p Path, fsys fs.FS, open func(context.Context, io.Reader) (T, error)) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, context.Cause(ctx)
	}
	f, err := p.Open(fsys)
	if err != nil {
		return zero, err
	}
	out, err := open(ctx, f)
	if err != nil {
		f.Close()
		return zero, err
	}
	return out, nil
}
