//go:build linux || windows

package wim

import (
	"context"
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
// Lookup walks the image's own directories. It does not keep an index.
type FS struct {
	r      *winwim.Reader
	root   entry
	closed bool
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Images lists the images in r. r must be an [io.ReaderAt].
func Images(ctx context.Context, r io.Reader) ([]Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	ra, err := lewfs.ReaderAt("images", r)
	if err != nil {
		return nil, err
	}
	rd, err := winwim.NewReader(lewfs.ContextReaderAt(ctx, ra))
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
func Open(ctx context.Context, r io.Reader, image int) (*FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	rd, err := winwim.NewReader(lewfs.ContextReaderAt(ctx, ra))
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
	return &FS{r: rd, root: wimNode{f: rootFile}}, nil
}

// entry is one directory the image already has.
type entry interface {
	lewfs.Entry
	info() fs.FileInfo
	open() (fs.File, error)
	children() ([]entry, error)
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
