package tar

import (
	stdtar "archive/tar"
	"io"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
)

// Files yields members of an uncompressed tar from r.
//
// Compressed wrappers are not unwrapped. Use [Open] for that.
//
// When r is an [io.ReaderAt], each regular file's Reader is an
// [io.SectionReader] at the member offset, so [lewfs.File.Open]
// stays valid after the iterator moves on. On a stream, Reader is
// the live [archive/tar.Reader] and is valid only until the next
// yield.
func Files(r io.Reader) lewfs.Files {
	return func(yield func(lewfs.File, error) bool) {
		var cr *cursor
		var tr *stdtar.Reader
		if ra, ok := r.(io.ReaderAt); ok {
			cr = &cursor{ra: ra}
			tr = stdtar.NewReader(cr)
		} else {
			tr = stdtar.NewReader(r)
		}
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(lewfs.File{}, err)
				return
			}
			if hdr.Typeflag == stdtar.TypeXGlobalHeader {
				continue
			}
			name, err := cleanName(hdr.Name)
			if err != nil {
				yield(lewfs.File{}, err)
				return
			}
			if name == "." {
				continue
			}
			info := hdr.FileInfo()
			f := lewfs.File{
				Name:    path.New(name),
				Mode:    info.Mode(),
				Size:    hdr.Size,
				ModTime: hdr.ModTime,
			}
			switch {
			case info.IsDir():
				f.Size = 0
			case info.Mode().IsRegular():
				if cr != nil {
					f.Reader = io.NewSectionReader(cr.ra, cr.off, hdr.Size)
				} else {
					f.Reader = tr
				}
			default:
				continue
			}
			if !yield(f, nil) {
				return
			}
		}
	}
}
