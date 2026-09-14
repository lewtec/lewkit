// Package fs is shared errors and helpers for x/fs/* adapters.
//
// Adapters take an [io.Reader] and call [ReaderAt]. A missing
// [io.ReaderAt] is [ErrNeedReadAt]. Zip also needs [Size].
// They do not spool.
//
// [File] and [Files] are a container listing. Sequential formats
// (tar) walk member headers. Formats with a table of contents walk
// that table. [New] indexes a listing into a read-only [io/fs.FS].
package fs

import (
	"errors"
	"io"
	iofs "io/fs"
)

// ErrNeedReadAt is [ReaderAt] on a reader that is not an [io.ReaderAt].
var ErrNeedReadAt = errors.New("need io.ReaderAt")

// ErrNeedSize is [Size] on a reader with no size.
var ErrNeedSize = errors.New("need size")

// ReaderAt returns r as an [io.ReaderAt], or [ErrNeedReadAt].
func ReaderAt(op string, r io.Reader) (io.ReaderAt, error) {
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, &iofs.PathError{Op: op, Path: "", Err: ErrNeedReadAt}
	}
	return ra, nil
}

type sizer interface {
	Size() int64
}

type stater interface {
	Stat() (iofs.FileInfo, error)
}

// Size returns the length of r.
// It tries Size, then Stat, then Seek to the end.
func Size(op string, r io.Reader) (int64, error) {
	if s, ok := r.(sizer); ok {
		return s.Size(), nil
	}
	if s, ok := r.(stater); ok {
		fi, err := s.Stat()
		if err == nil {
			return fi.Size(), nil
		}
	}
	if s, ok := r.(io.Seeker); ok {
		cur, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, &iofs.PathError{Op: op, Path: "", Err: err}
		}
		end, err := s.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, &iofs.PathError{Op: op, Path: "", Err: err}
		}
		if _, err := s.Seek(cur, io.SeekStart); err != nil {
			return 0, &iofs.PathError{Op: op, Path: "", Err: err}
		}
		return end, nil
	}
	return 0, &iofs.PathError{Op: op, Path: "", Err: ErrNeedSize}
}
