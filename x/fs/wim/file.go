//go:build linux || windows

package wim

import (
	"io"
	"io/fs"

	winwim "github.com/Microsoft/go-winio/wim"
)

func (n *dnode) open() (fs.File, error) {
	if n.IsDir() {
		return n.OpenDir(infoOf), nil
	}
	wf := n.Payload()
	if wf == nil {
		return nil, &fs.PathError{Op: "open", Path: n.Name(), Err: fs.ErrInvalid}
	}
	return &file{info: n.info(), wf: wf}, nil
}

func (n *dnode) readDir() []fs.DirEntry {
	return n.Entries(infoOf)
}

type file struct {
	info fs.FileInfo
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
