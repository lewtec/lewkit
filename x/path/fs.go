package path

import (
	"errors"
	"io/fs"
	"iter"
	"os"
)

// Root is an [os.Root] that is also an [io/fs.FS] with write methods.
type Root struct {
	r *os.Root
}

// Open opens dir with [os.OpenRoot]. dir is an OS path, not a [Path].
func Open(dir string) (*Root, error) {
	if dir == "" {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: ErrEmptyPath}
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		if st, stErr := os.Lstat(dir); stErr == nil && !st.IsDir() {
			return nil, &fs.PathError{Op: "open", Path: dir, Err: ErrNotDir}
		}
		return nil, err
	}
	return &Root{r: r}, nil
}

func (rt *Root) Close() error { return rt.r.Close() }

func (rt *Root) Name() string { return rt.r.Name() }

func (rt *Root) Open(name string) (fs.File, error) {
	return rt.r.Open(name)
}

func (rt *Root) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return rt.r.OpenFile(name, flag, perm)
}

func (rt *Root) Create(name string) (fs.File, error) {
	return rt.r.Create(name)
}

func (rt *Root) ReadFile(name string) ([]byte, error) {
	return rt.r.ReadFile(name)
}

func (rt *Root) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return rt.r.WriteFile(name, data, perm)
}

func (rt *Root) Stat(name string) (fs.FileInfo, error) {
	return rt.r.Stat(name)
}

func (rt *Root) Lstat(name string) (fs.FileInfo, error) {
	return rt.r.Lstat(name)
}

func (rt *Root) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(rt.r.FS(), name)
}

func (rt *Root) ReadLink(name string) (string, error) {
	return rt.r.Readlink(name)
}

func (rt *Root) Mkdir(name string, perm fs.FileMode) error {
	return rt.r.Mkdir(name, perm)
}

func (rt *Root) MkdirAll(name string, perm fs.FileMode) error {
	return rt.r.MkdirAll(name, perm)
}

func (rt *Root) Remove(name string) error { return rt.r.Remove(name) }

func (rt *Root) RemoveAll(name string) error { return rt.r.RemoveAll(name) }

func (rt *Root) Rename(oldname, newname string) error {
	return rt.r.Rename(oldname, newname)
}

func (rt *Root) Symlink(oldname, newname string) error {
	return rt.r.Symlink(oldname, newname)
}

func (rt *Root) Link(oldname, newname string) error {
	return rt.r.Link(oldname, newname)
}

func (rt *Root) Chmod(name string, mode fs.FileMode) error {
	return rt.r.Chmod(name, mode)
}

func (rt *Root) OpenRoot(name string) (*Root, error) {
	r, err := rt.r.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return &Root{r: r}, nil
}

func (rt *Root) Sub(dir string) (fs.FS, error) {
	return rt.OpenRoot(dir)
}

var (
	_ fs.FS         = (*Root)(nil)
	_ fs.ReadFileFS = (*Root)(nil)
	_ fs.StatFS     = (*Root)(nil)
	_ fs.ReadDirFS  = (*Root)(nil)
	_ fs.ReadLinkFS = (*Root)(nil)
	_ fs.SubFS      = (*Root)(nil)
)

func readOnly(op, name string) error {
	return &fs.PathError{Op: op, Path: name, Err: ErrReadOnly}
}

func missing(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// Open opens p on fsys.
func (p Path) Open(fsys fs.FS) (fs.File, error) {
	return fsys.Open(p.s)
}

type openFileFS interface {
	OpenFile(string, int, fs.FileMode) (fs.File, error)
}

// OpenFile opens p with flag and perm when fsys implements OpenFile.
func (p Path) OpenFile(fsys fs.FS, flag int, perm fs.FileMode) (fs.File, error) {
	if o, ok := fsys.(openFileFS); ok {
		return o.OpenFile(p.s, flag, perm)
	}
	return nil, readOnly("open", p.s)
}

type createFS interface {
	Create(string) (fs.File, error)
}

// Create creates or truncates p.
func (p Path) Create(fsys fs.FS) (fs.File, error) {
	if c, ok := fsys.(createFS); ok {
		return c.Create(p.s)
	}
	return nil, readOnly("create", p.s)
}

// ReadFile reads p.
func (p Path) ReadFile(fsys fs.FS) ([]byte, error) {
	return fs.ReadFile(fsys, p.s)
}

type writeFileFS interface {
	WriteFile(string, []byte, fs.FileMode) error
}

// WriteFile writes data to p.
func (p Path) WriteFile(fsys fs.FS, data []byte, perm fs.FileMode) error {
	if w, ok := fsys.(writeFileFS); ok {
		return w.WriteFile(p.s, data, perm)
	}
	return readOnly("write", p.s)
}

// Stat stats p.
func (p Path) Stat(fsys fs.FS) (fs.FileInfo, error) {
	return fs.Stat(fsys, p.s)
}

// Lstat stats p without following a symlink.
func (p Path) Lstat(fsys fs.FS) (fs.FileInfo, error) {
	return fs.Lstat(fsys, p.s)
}

// ReadDir lists p.
func (p Path) ReadDir(fsys fs.FS) ([]fs.DirEntry, error) {
	return fs.ReadDir(fsys, p.s)
}

// ReadLink returns the symlink target of p as a Path.
func (p Path) ReadLink(fsys fs.FS) (Path, error) {
	s, err := fs.ReadLink(fsys, p.s)
	if err != nil {
		return Path{}, err
	}
	return Path{s: s}, nil
}

// WalkDir walks p.
func (p Path) WalkDir(fsys fs.FS, fn fs.WalkDirFunc) error {
	return fs.WalkDir(fsys, p.s, fn)
}

// Walk yields p and every name under it.
func (p Path) Walk(fsys fs.FS) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		err := fs.WalkDir(fsys, p.s, func(name string, _ fs.DirEntry, err error) error {
			if err != nil {
				if !yield(Path{s: name}, err) {
					return fs.SkipAll
				}
				return err
			}
			if !yield(Path{s: name}, nil) {
				return fs.SkipAll
			}
			return nil
		})
		if err != nil {
			return
		}
	}
}

// IterDir yields the children of p.
func (p Path) IterDir(fsys fs.FS) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		ents, err := fs.ReadDir(fsys, p.s)
		if err != nil {
			yield(Path{}, err)
			return
		}
		for _, e := range ents {
			if !yield(p.Join(e.Name()), nil) {
				return
			}
		}
	}
}

type mkdirFS interface {
	Mkdir(string, fs.FileMode) error
}

// Mkdir creates p.
func (p Path) Mkdir(fsys fs.FS, perm fs.FileMode) error {
	if m, ok := fsys.(mkdirFS); ok {
		return m.Mkdir(p.s, perm)
	}
	return readOnly("mkdir", p.s)
}

type mkdirAllFS interface {
	MkdirAll(string, fs.FileMode) error
}

// MkdirAll creates p and any missing parents.
func (p Path) MkdirAll(fsys fs.FS, perm fs.FileMode) error {
	if m, ok := fsys.(mkdirAllFS); ok {
		return m.MkdirAll(p.s, perm)
	}
	return readOnly("mkdirall", p.s)
}

type removeFS interface {
	Remove(string) error
}

// Remove removes p.
func (p Path) Remove(fsys fs.FS) error {
	if r, ok := fsys.(removeFS); ok {
		return r.Remove(p.s)
	}
	return readOnly("remove", p.s)
}

type removeAllFS interface {
	RemoveAll(string) error
}

// RemoveAll removes p and its children.
func (p Path) RemoveAll(fsys fs.FS) error {
	if r, ok := fsys.(removeAllFS); ok {
		return r.RemoveAll(p.s)
	}
	return readOnly("removeall", p.s)
}

type renameFS interface {
	Rename(oldname, newname string) error
}

// Rename moves p to dest on the same filesystem.
func (p Path) Rename(fsys fs.FS, dest Path) error {
	if r, ok := fsys.(renameFS); ok {
		return r.Rename(p.s, dest.s)
	}
	return readOnly("rename", p.s)
}

type symlinkFS interface {
	Symlink(oldname, newname string) error
}

// Symlink creates p as a symlink to target.
func (p Path) Symlink(fsys fs.FS, target Path) error {
	if s, ok := fsys.(symlinkFS); ok {
		return s.Symlink(target.s, p.s)
	}
	return readOnly("symlink", p.s)
}

type linkFS interface {
	Link(oldname, newname string) error
}

// Hardlink creates p as a hard link to target.
func (p Path) Hardlink(fsys fs.FS, target Path) error {
	if l, ok := fsys.(linkFS); ok {
		return l.Link(target.s, p.s)
	}
	return readOnly("link", p.s)
}

type chmodFS interface {
	Chmod(string, fs.FileMode) error
}

// Chmod changes the mode of p.
func (p Path) Chmod(fsys fs.FS, mode fs.FileMode) error {
	if c, ok := fsys.(chmodFS); ok {
		return c.Chmod(p.s, mode)
	}
	return readOnly("chmod", p.s)
}

type openRootFS interface {
	OpenRoot(string) (*Root, error)
}

// OpenRoot opens p as a nested root when fsys is a [*Root].
func (p Path) OpenRoot(fsys fs.FS) (*Root, error) {
	if o, ok := fsys.(openRootFS); ok {
		return o.OpenRoot(p.s)
	}
	return nil, readOnly("openroot", p.s)
}

// Exists reports whether p is present. A missing name is false, nil.
func (p Path) Exists(fsys fs.FS) (bool, error) {
	_, err := fs.Stat(fsys, p.s)
	if err == nil {
		return true, nil
	}
	if missing(err) {
		return false, nil
	}
	return false, err
}

// IsDir reports whether p is a directory. A missing name is false, nil.
func (p Path) IsDir(fsys fs.FS) (bool, error) {
	st, err := fs.Stat(fsys, p.s)
	if err == nil {
		return st.IsDir(), nil
	}
	if missing(err) {
		return false, nil
	}
	return false, err
}

// IsFile reports whether p is a regular file. A missing name is false, nil.
func (p Path) IsFile(fsys fs.FS) (bool, error) {
	st, err := fs.Stat(fsys, p.s)
	if err == nil {
		return st.Mode().IsRegular(), nil
	}
	if missing(err) {
		return false, nil
	}
	return false, err
}

// IsSymlink reports whether p is a symlink. A missing name is false, nil.
func (p Path) IsSymlink(fsys fs.FS) (bool, error) {
	st, err := fs.Lstat(fsys, p.s)
	if err == nil {
		return st.Mode()&fs.ModeSymlink != 0, nil
	}
	if missing(err) {
		return false, nil
	}
	return false, err
}
