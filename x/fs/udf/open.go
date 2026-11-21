package udf

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	stdpath "path"

	"github.com/Xmister/udf"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// FS is a read-only UDF volume.
type FS struct {
	root *dnode
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Open reads a UDF volume from r. r must be an [io.ReaderAt].
func Open(ctx context.Context, r io.Reader) (out *FS, err error) {
	defer recovered(&err)
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	u, err := udf.NewUdfFromReader(lewfs.ContextReaderAt(ctx, ra))
	if err != nil {
		return nil, err
	}
	root := newDir(".")
	var walk func(*dnode, []udf.File, string) error
	walk = func(root *dnode, items []udf.File, prefix string) error {
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		for i := range items {
			if err := ctx.Err(); err != nil {
				return context.Cause(ctx)
			}
			item := &items[i]
			name := item.Name()
			if name == "" || name == "." || name == ".." {
				continue
			}
			p := name
			if prefix != "" {
				p = stdpath.Join(prefix, name)
			}
			p = stdpath.Clean(p)
			if !fs.ValidPath(p) {
				return &fs.PathError{Op: "open", Path: p, Err: fs.ErrInvalid}
			}
			if item.IsDir() {
				if err := root.add(p, item, true); err != nil {
					return err
				}
				if err := walk(root, item.ReadDir(), p); err != nil {
					return err
				}
				continue
			}
			if err := root.add(p, item, false); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root, u.ReadDir(nil), ""); err != nil {
		return nil, err
	}
	return &FS{root: root}, nil
}

func recovered(errp *error) {
	rec := recover()
	if rec == nil {
		return
	}
	*errp = recoveredError{rec}
}

type recoveredError struct{ v any }

func (e recoveredError) Error() string {
	if err, ok := e.v.(error); ok {
		return "udf: " + err.Error()
	}
	return "udf: " + fmt.Sprint(e.v)
}

func (e recoveredError) Unwrap() error {
	err, _ := e.v.(error)
	return err
}
