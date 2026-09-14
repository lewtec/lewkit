package fs

import (
	"cmp"
	"io"
	iofs "io/fs"
	"slices"
	"time"
)

func (n *dnode) readDir() []iofs.DirEntry {
	out := make([]iofs.DirEntry, 0, len(n.kids))
	for _, k := range n.kids {
		out = append(out, iofs.FileInfoToDirEntry(k.info()))
	}
	slices.SortFunc(out, func(a, b iofs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

type dirFile struct {
	n     *dnode
	infos []iofs.DirEntry
	off   int
}

func (d *dirFile) Stat() (iofs.FileInfo, error) { return d.n.info(), nil }

func (d *dirFile) Read([]byte) (int, error) {
	return 0, &iofs.PathError{Op: "read", Path: d.n.name, Err: iofs.ErrInvalid}
}

func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]iofs.DirEntry, error) {
	if d.off >= len(d.infos) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := d.infos[d.off:]
		d.off = len(d.infos)
		return out, nil
	}
	end := d.off + n
	if end > len(d.infos) {
		end = len(d.infos)
	}
	out := d.infos[d.off:end]
	d.off = end
	return out, nil
}

type fileInfo struct {
	name string
	size int64
	mode iofs.FileMode
	mod  time.Time
}

func (i fileInfo) Name() string        { return i.name }
func (i fileInfo) Size() int64         { return i.size }
func (i fileInfo) Mode() iofs.FileMode { return i.mode }
func (i fileInfo) ModTime() time.Time  { return i.mod }
func (i fileInfo) IsDir() bool         { return i.mode.IsDir() }
func (i fileInfo) Sys() any            { return nil }
