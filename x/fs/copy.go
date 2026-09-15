package fs

import (
	"context"
	"errors"
	"io"
	iofs "io/fs"
	"os"

	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/path/pick"
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

var errStop = errors.New("stop")

// Walk yields [Files] from fsys. pred is applied during the walk so a
// pruned directory is never opened. A nil pred yields every name,
// including empty directories.
func Walk(fsys iofs.FS, pred pick.Predicate) Files {
	return func(yield func(File, error) bool) {
		err := iofs.WalkDir(fsys, ".", func(name string, d iofs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if name == "." {
				if !d.IsDir() {
					return &iofs.PathError{Op: "copy", Path: name, Err: iofs.ErrInvalid}
				}
				return nil
			}
			p := path.New(name)
			if d.Type()&iofs.ModeSymlink != 0 {
				info, err := d.Info()
				if err != nil {
					return err
				}
				if !yield(File{Name: p, Mode: info.Mode()}, nil) {
					return errStop
				}
				return nil
			}
			if d.IsDir() {
				if !pred.Accept(p, true) {
					return iofs.SkipDir
				}
				if pred != nil {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				if !yield(File{Name: p, Mode: info.Mode()}, nil) {
					return errStop
				}
				return nil
			}
			if !pred.Accept(p, false) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			r, err := fsys.Open(name)
			if err != nil {
				return err
			}
			ok := yield(File{
				Name:    p,
				Mode:    info.Mode(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Reader:  r,
			}, nil)
			r.Close()
			if !ok {
				return errStop
			}
			return nil
		})
		if err != nil && !errors.Is(err, errStop) {
			yield(File{}, err)
		}
	}
}

// Filter applies pred to a listing. A nil pred is the listing unchanged.
// Names under a pruned directory are dropped even if that directory
// never appeared as its own member.
func Filter(files Files, pred pick.Predicate) Files {
	if pred == nil {
		return files
	}
	return func(yield func(File, error) bool) {
		for f, err := range files {
			if err != nil {
				yield(File{}, err)
				return
			}
			if f.Mode.IsDir() || pruned(f.Name, pred) || !pred.Accept(f.Name, false) {
				continue
			}
			if !yield(f, nil) {
				return
			}
		}
	}
}

func pruned(p path.Path, pred pick.Predicate) bool {
	for cur := p.Parent(); cur.String() != "." && cur.String() != ""; cur = cur.Parent() {
		if !pred.Accept(cur, true) {
			return true
		}
	}
	return false
}

// Copy writes files into dest, like rsync from/ to.
// Dest root must already exist; "." is not created.
//
// Each member body is consumed before the listing moves on, so a
// stream [File.Reader] stays valid. Directory members are created;
// file parents are created as needed.
//
// An existing dest file is [iofs.ErrExist]. A symlink is [iofs.ErrInvalid].
//
// Files are created with mode 0o666 plus source execute bits.
// Directories are created with mode 0o777.
//
// A tree is [Walk]. A tar stream is [github.com/lewtec/lewkit/x/fs/tar.Files].
func Copy(ctx context.Context, dest DestFS, files Files) error {
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
			if err := dest.MkdirAll(name, 0o777); err != nil {
				return err
			}
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
