//go:build linux || windows

package wim

import (
	"cmp"
	"io"
	"io/fs"
	"slices"
	"time"

	winwim "github.com/Microsoft/go-winio/wim"
)

func (n *dnode) open() (fs.File, error) {
	if n.dir {
		return &dirFile{n: n, infos: n.readDir()}, nil
	}
	if n.wf == nil {
		return nil, &fs.PathError{Op: "open", Path: n.name, Err: fs.ErrInvalid}
	}
	return &file{info: n.info(), wf: n.wf}, nil
}

func (n *dnode) readDir() []fs.DirEntry {
	out := make([]fs.DirEntry, 0, len(n.kids))
	for _, k := range n.kids {
		out = append(out, fs.FileInfoToDirEntry(k.info()))
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out
}

type file struct {
	info fileInfo
	wf   *winwim.File
	r    io.ReadCloser
}

var _ fs.File = (*file)(nil)

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *file) Read(p []byte) (int, error) {
	if f.r == nil {
		r, err := f.wf.Open()
		if err != nil {
			return 0, err
		}
		f.r = r
	}
	return f.r.Read(p)
}

func (f *file) Close() error {
	if f.r == nil {
		return nil
	}
	err := f.r.Close()
	f.r = nil
	return err
}

type dirFile struct {
	n     *dnode
	infos []fs.DirEntry
	off   int
}

func (d *dirFile) Stat() (fs.FileInfo, error) { return d.n.info(), nil }

func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.n.name, Err: fs.ErrInvalid}
}

func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
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
	mode fs.FileMode
	mod  time.Time
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return i.mod }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return nil }
