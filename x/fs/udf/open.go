package udf

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/Xmister/udf"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// FS is a read-only UDF volume.
// Lookup walks the volume's own directories. It does not keep an index.
type FS struct {
	root entry
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
	return &FS{root: udfNode{vol: u}}, nil
}

// entry is one directory the volume already has.
type entry interface {
	lewfs.Entry
	info() fs.FileInfo
	open() (fs.File, error)
	children() ([]entry, error)
}

type udfNode struct {
	vol  *udf.Udf
	file *udf.File
}

func (n udfNode) Name() string {
	if n.file == nil {
		return "."
	}
	return n.file.Name()
}

func (n udfNode) IsDir() bool {
	return n.file == nil || n.file.IsDir()
}

func (n udfNode) children() ([]entry, error) {
	var items []udf.File
	if n.file == nil {
		items = n.vol.ReadDir(nil)
	} else {
		items = n.file.ReadDir()
	}
	out := make([]entry, 0, len(items))
	for i := range items {
		item := items[i]
		name := item.Name()
		if name == "" || name == "." || name == ".." {
			continue
		}
		out = append(out, udfNode{vol: n.vol, file: &item})
	}
	return out, nil
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
