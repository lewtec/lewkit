package fs

import (
	"io"
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
	// Reader is the member body. It is nil for a directory.
	// If Reader is an [io.ReaderAt], [File.Open] can be called
	// more than once. A stream body is valid only until the
	// listing moves on.
	Reader io.Reader
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

// Open returns a reader for the member body.
func (f File) Open() (iofs.File, error) {
	if f.Mode.IsDir() || f.Reader == nil {
		return nil, &iofs.PathError{Op: "open", Path: f.Name.String(), Err: iofs.ErrInvalid}
	}
	info := f.info()
	if ra, ok := f.Reader.(io.ReaderAt); ok {
		return Section(info, ra, 0, f.Size), nil
	}
	return &file{fileInfo: info, r: f.Reader}, nil
}

func (f File) info() fileInfo {
	name := f.Name.Name()
	mode := f.Mode
	if mode == 0 {
		mode = 0o444
	}
	if f.Mode.IsDir() && mode&iofs.ModeDir == 0 {
		mode |= iofs.ModeDir | 0o555
	}
	return fileInfo{name: name, size: f.Size, mode: mode, mod: f.ModTime}
}

var (
	_ iofs.File   = (*file)(nil)
	_ iofs.File   = (*sectionFile)(nil)
	_ io.ReaderAt = (*sectionFile)(nil)
	_ io.Seeker   = (*sectionFile)(nil)
)

type file struct {
	fileInfo
	r io.Reader
}

func (f *file) Stat() (iofs.FileInfo, error) { return f.fileInfo, nil }

func (f *file) Read(p []byte) (int, error) { return f.r.Read(p) }

func (*file) Close() error { return nil }

// Section is a read-only [io/fs.File] over [io.NewSectionReader].
func Section(info iofs.FileInfo, ra io.ReaderAt, off, n int64) iofs.File {
	return &sectionFile{info: info, SectionReader: io.NewSectionReader(ra, off, n)}
}

type sectionFile struct {
	info iofs.FileInfo
	*io.SectionReader
}

func (f *sectionFile) Stat() (iofs.FileInfo, error) { return f.info, nil }

func (*sectionFile) Close() error { return nil }
