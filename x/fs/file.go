package fs

import (
	iofs "io/fs"
	"iter"
	"time"

	"github.com/lewtec/lewkit/x/path"
)

// File is one member of a container.
type File struct {
	// Name is the [io/fs] path of the member.
	Name path.Path
	// Mode is the member mode.
	Mode iofs.FileMode
	// Size is the member size in bytes.
	Size int64
	// ModTime is the member modification time.
	ModTime time.Time
	// Open returns the member. After the iterator moves on, Open
	// stays valid only when the adapter recorded a stable handle
	// (a ReaderAt offset, a zip.File, ...). A stream adapter binds
	// Open to the live sequential reader; that is valid only until
	// the next yield. Open is nil for a directory.
	Open func() (iofs.File, error)
}

// Files is a container listing.
type Files = iter.Seq2[File, error]

// Names yields the name of each file.
func Names(files Files) iter.Seq2[path.Path, error] {
	return func(yield func(path.Path, error) bool) {
		for f, err := range files {
			if err != nil {
				yield(path.Path{}, err)
				return
			}
			if !yield(f.Name, nil) {
				return
			}
		}
	}
}
