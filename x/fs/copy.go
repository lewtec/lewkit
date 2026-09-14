package fs

import (
	"context"
	"io"
	iofs "io/fs"
	"os"

	"github.com/lewtec/lewkit/x/path"
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
// keep is called for each name except ".". A nil keep keeps everything.
// A false file is skipped. A false directory is still walked, so a
// glob like **/*.go can match children; the directory is created only
// if a kept file needs it or keep is true.
//
// An existing dest file is [iofs.ErrExist]. A symlink is [iofs.ErrInvalid].
//
// Files are created with mode 0o666 plus source execute bits.
// Directories are created with mode 0o777.
//
// Copy walks an [io/fs.FS]. For a sequential listing such as
// [github.com/lewtec/lewkit/x/fs/tar.Files], use [CopyFiles].
func Copy(ctx context.Context, src iofs.FS, dest DestFS, keep func(path.Path) bool) error {
	w := destWriter{ctx: ctx, dest: dest}
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
		p := path.New(name)
		if d.IsDir() {
			return w.dir(p, keep)
		}
		if keep != nil && !keep(p) {
			return nil
		}
		if err := w.parents(p); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		r, err := src.Open(name)
		if err != nil {
			return err
		}
		defer r.Close()
		return w.file(name, info.Mode().Perm(), r)
	})
}

// CopyFiles writes a [Files] listing into dest, in listing order.
// Dest root must already exist; "." is not created.
//
// Each member body is consumed before the listing moves on, so a
// stream [File.Reader] stays valid. keep matches [Copy].
func CopyFiles(ctx context.Context, files Files, dest DestFS, keep func(path.Path) bool) error {
	w := destWriter{ctx: ctx, dest: dest}
	for f, err := range files {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		name := f.Name.String()
		if name == "." || name == "" {
			continue
		}
		if !f.Name.Valid() {
			return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
		}
		if f.Mode&iofs.ModeSymlink != 0 {
			return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
		}
		if f.Mode.IsDir() {
			if err := w.dir(f.Name, keep); err != nil {
				return err
			}
			continue
		}
		if keep != nil && !keep(f.Name) {
			continue
		}
		if err := w.parents(f.Name); err != nil {
			return err
		}
		if f.Reader == nil {
			return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
		}
		if err := w.file(name, f.Mode.Perm(), f.Reader); err != nil {
			return err
		}
	}
	return nil
}

type destWriter struct {
	ctx  context.Context
	dest DestFS
}

func (w destWriter) dir(p path.Path, keep func(path.Path) bool) error {
	if keep != nil && !keep(p) {
		return nil
	}
	return w.dest.MkdirAll(p.String(), 0o777)
}

func (w destWriter) parents(p path.Path) error {
	parent := p.Parent().String()
	if parent == "." {
		return nil
	}
	return w.dest.MkdirAll(parent, 0o777)
}

func (w destWriter) file(name string, perm iofs.FileMode, r io.Reader) error {
	mode := iofs.FileMode(0o666) | perm&0o111
	f, err := w.dest.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	out, ok := f.(io.Writer)
	if !ok {
		return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
	}
	_, err = io.Copy(out, ctxReader{ctx: w.ctx, r: r})
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
