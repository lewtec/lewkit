package squashfs

import (
	"cmp"
	"io/fs"
	"slices"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (fs.File, error) {
	if name == "." || name == "" {
		return d.openDir(".")
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	st, err := d.r.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if st.IsDir() {
		return d.openDir(name)
	}
	f, err := d.r.Open(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &file{File: f}, nil
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name != "." && name != "" && !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if name == "" {
		name = "."
	}
	st, err := d.r.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	if !st.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	ents, err := d.r.ReadDir(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	slices.SortFunc(ents, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return ents, nil
}

// ReadFile implements [fs.ReadFileFS].
func (d *FS) ReadFile(name string) ([]byte, error) {
	if name != "." && name != "" && !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	st, err := d.r.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	if st.IsDir() {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	return d.r.ReadFile(name)
}

// Stat implements [fs.StatFS].
func (d *FS) Stat(name string) (fs.FileInfo, error) {
	if name == "." || name == "" {
		return d.r.Stat(".")
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	st, err := d.r.Stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return st, nil
}

func (d *FS) openDir(name string) (fs.File, error) {
	ents, err := d.ReadDir(name)
	if err != nil {
		return nil, err
	}
	st, err := d.Stat(name)
	if err != nil {
		return nil, err
	}
	return &dirFile{info: st, ents: ents}, nil
}
