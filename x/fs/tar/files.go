package tar

import (
	stdtar "archive/tar"
	"io"
	"io/fs"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
)

// Files yields members of an uncompressed tar from r.
//
// Compressed wrappers are not unwrapped. Use [Open] for that.
//
// When r is an [io.ReaderAt], each regular file's Open stays valid
// after the iterator moves on (a [io.SectionReader] at the member
// offset). On a stream, Open reads the live [archive/tar.Reader]
// and is valid only until the next yield.
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
				f.Open = nil
			case info.Mode().IsRegular():
				base := f.Name.Name()
				mode := info.Mode()
				mod := hdr.ModTime
				size := hdr.Size
				if cr != nil {
					off := cr.off
					ra := cr.ra
					f.Open = func() (fs.File, error) {
						return &file{
							info: fileInfo{name: base, size: size, mode: mode, mod: mod},
							r:    io.NewSectionReader(ra, off, size),
						}, nil
					}
				} else {
					f.Open = func() (fs.File, error) {
						return &streamFile{
							info: fileInfo{name: base, size: size, mode: mode, mod: mod},
							r:    tr,
						}, nil
					}
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

type streamFile struct {
	info fileInfo
	r    io.Reader
}

func (f *streamFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *streamFile) Read(p []byte) (int, error) { return f.r.Read(p) }

func (f *streamFile) Close() error { return nil }
