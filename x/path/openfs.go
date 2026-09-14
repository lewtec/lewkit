package path

import (
	"io"
	"io/fs"
)

// OpenFS opens p on fsys and passes the file to open.
// If open fails, the file is closed. On success the file stays open
// for the returned value (it must be an [io.ReaderAt] when the
// adapter needs random access).
func OpenFS[T any](p Path, fsys fs.FS, open func(io.Reader) (T, error)) (T, error) {
	var zero T
	f, err := p.Open(fsys)
	if err != nil {
		return zero, err
	}
	out, err := open(f)
	if err != nil {
		f.Close()
		return zero, err
	}
	return out, nil
}
