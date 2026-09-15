//go:build linux || windows

package wim

import (
	"errors"
	"io"
	"io/fs"

	winwim "github.com/Microsoft/go-winio/wim"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// ErrInvalidImage is [Open] with an image index that is not in the WIM.
var ErrInvalidImage = errors.New("invalid image")

// Info is one image in a WIM.
type Info struct {
	// Index is the 1-based image index.
	Index int
	// Name is the image name from the WIM XML.
	Name string
}

// FS is a read-only WIM image.
type FS struct {
	r      *winwim.Reader
	root   *dnode
	closed bool
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Images lists the images in r. r must be an [io.ReaderAt].
func Images(r io.Reader) ([]Info, error) {
	ra, err := lewfs.ReaderAt("images", r)
	if err != nil {
		return nil, err
	}
	rd, err := winwim.NewReader(ra)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	out := make([]Info, len(rd.Image))
	for i, img := range rd.Image {
		out[i] = Info{Index: i + 1, Name: img.Name}
	}
	return out, nil
}

// Open reads image from r. image is 1-based. r must be an [io.ReaderAt].
func Open(r io.Reader, image int) (*FS, error) {
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	rd, err := winwim.NewReader(ra)
	if err != nil {
		return nil, err
	}
	if image < 1 || image > len(rd.Image) {
		rd.Close()
		return nil, &fs.PathError{Op: "open", Path: "", Err: ErrInvalidImage}
	}
	rootFile, err := rd.Image[image-1].Open()
	if err != nil {
		rd.Close()
		return nil, err
	}
	root := newDir(".")
	if err := walkWIM(root, rootFile, ""); err != nil {
		rd.Close()
		return nil, err
	}
	return &FS{r: rd, root: root}, nil
}

// Close releases the WIM reader.
func (d *FS) Close() error {
	if d.closed {
		return nil
	}
	d.closed = true
	if d.r == nil {
		return nil
	}
	err := d.r.Close()
	d.r = nil
	return err
}

func walkWIM(root *dnode, dir *winwim.File, prefix string) error {
	ents, err := dir.Readdir()
	if err != nil {
		return err
	}
	for _, e := range ents {
		name := e.Name
		if name == "" || name == "." || name == ".." {
			continue
		}
		p := name
		if prefix != "" {
			p = prefix + "/" + name
		}
		if !fs.ValidPath(p) {
			return &fs.PathError{Op: "open", Path: p, Err: fs.ErrInvalid}
		}
		if e.IsDir() {
			if err := root.Add(p, e, true); err != nil {
				return err
			}
			if err := walkWIM(root, e, p); err != nil {
				return err
			}
			continue
		}
		if err := root.Add(p, e, false); err != nil {
			return err
		}
	}
	return nil
}
