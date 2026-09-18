package zip

import (
	stdzip "archive/zip"
	"io"
	"io/fs"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// Open implements [fs.FS].
func (d *FS) Open(name string) (fs.File, error) {
	if err := lewfs.CheckName("open", name); err != nil {
		return nil, err
	}
	if zf := d.lookup(name); zf != nil && !zf.FileInfo().IsDir() && zf.Method == stdzip.Store {
		return openStored(d.ra, zf)
	}
	return d.r.Open(name)
}

// ReadDir implements [fs.ReadDirFS].
func (d *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(d.r, name)
}

// ReadFile implements [fs.ReadFileFS].
func (d *FS) ReadFile(name string) ([]byte, error) {
	if zf := d.lookup(name); zf != nil && !zf.FileInfo().IsDir() {
		f, err := zf.Open()
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}
	return fs.ReadFile(d.r, name)
}

// Stat implements [fs.StatFS].
func (d *FS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(d.r, name)
}
