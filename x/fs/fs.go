// Package fs is shared errors and helpers for x/fs/* adapters.
//
// Adapters take an [io.Reader] and call [ReaderAt]. A missing
// [io.ReaderAt] is [ErrNeedReadAt]. Zip also needs [Size].
// They do not spool.
//
// [File] is one listing member: name plus a Reader for the body.
// [File.Open] reads that body. Sequential formats (tar) walk member
// headers. Formats with a table of contents walk that table.
// [New] indexes a listing into a read-only [io/fs.FS].
// [Copy] writes a [Files] listing into a dest, like rsync from/ to.
// [Walk] turns an [io/fs.FS] into a listing. [Filter] applies a
// [github.com/lewtec/lewkit/x/path/pick.Predicate].
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

// DirEntries pages ents starting at *off. n follows [io/fs.ReadDirFile]:
// n <= 0 returns the rest; n > 0 at the end is [io.EOF].
func DirEntries(ents []iofs.DirEntry, off *int, n int) ([]iofs.DirEntry, error) {
	if *off >= len(ents) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := ents[*off:]
		*off = len(ents)
		return out, nil
	}
	end := *off + n
	if end > len(ents) {
		end = len(ents)
	}
	out := ents[*off:end]
	*off = end
	return out, nil
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
