package udf

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	stdpath "path"

	"github.com/Xmister/udf"
)

// ErrNeedReadAt is [Open] on a reader that is not an [io.ReaderAt].
var ErrNeedReadAt = errors.New("need io.ReaderAt")

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
func Open(r io.Reader) (out *FS, err error) {
	defer recovered(&err)
	ra, err := readerAt("open", r)
	if err != nil {
		return nil, err
	}
	u, err := udf.NewUdfFromReader(ra)
	if err != nil {
		return nil, err
	}
	root := newDir(".")
	if err := walkUDF(root, u.ReadDir(nil), ""); err != nil {
		return nil, err
	}
	return &FS{root: root}, nil
}

func walkUDF(root *dnode, items []udf.File, prefix string) error {
	for i := range items {
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
			if err := walkUDF(root, item.ReadDir(), p); err != nil {
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

func readerAt(op string, r io.Reader) (io.ReaderAt, error) {
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, &fs.PathError{Op: op, Path: "", Err: ErrNeedReadAt}
	}
	return ra, nil
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
