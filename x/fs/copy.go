package fs

import (
	"context"
	"io"
	iofs "io/fs"
	"os"
)

// OpenFileFS creates or truncates a file.
type OpenFileFS interface {
	OpenFile(string, int, iofs.FileMode) (iofs.File, error)
}

// MkdirAllFS creates a directory and any missing parents.
type MkdirAllFS interface {
	MkdirAll(string, iofs.FileMode) error
}

// DestFS is a writable destination for [Copy].
type DestFS interface {
	OpenFileFS
	MkdirAllFS
}

// Copy writes the contents of src into dest, like rsync from/ to.
// Dest root must already exist; "." is not created.
//
// An existing dest file is [iofs.ErrExist]. A symlink is [iofs.ErrInvalid].
//
// Files are created with mode 0o666 plus source execute bits.
// Directories are created with mode 0o777.
func Copy(ctx context.Context, src iofs.FS, dest DestFS) error {
	c := copier{ctx: ctx, src: src, dest: dest}
	return iofs.WalkDir(src, ".", func(name string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		if name == "." {
			if !d.IsDir() {
				return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
			}
			return nil
		}
		if d.Type()&iofs.ModeSymlink != 0 {
			return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
		}
		if d.IsDir() {
			return dest.MkdirAll(name, 0o777)
		}
		return c.file(name, d)
	})
}

type copier struct {
	ctx  context.Context
	src  iofs.FS
	dest DestFS
}

func (c copier) file(name string, d iofs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}
	mode := iofs.FileMode(0o666) | info.Mode().Perm()&0o111
	r, err := c.src.Open(name)
	if err != nil {
		return err
	}
	defer r.Close()
	f, err := c.dest.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	w, ok := f.(io.Writer)
	if !ok {
		return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
	}
	_, err = io.Copy(w, ctxReader{ctx: c.ctx, r: r})
	return err
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (r ctxReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, context.Cause(r.ctx)
	}
	return r.r.Read(p)
}
