//go:build linux || windows

package wim

import (
	"io"
	"io/fs"

	winwim "github.com/Microsoft/go-winio/wim"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

func openNode(n *dnode) (fs.File, error) {
	if n.Dir {
		return &dtree.DirFile{Info: nodeInfo(n), Ents: n.Entries(nodeInfo)}, nil
	}
	if n.Val == nil {
		return nil, &fs.PathError{Op: "open", Path: n.Name, Err: fs.ErrInvalid}
	}
	return &file{info: nodeInfo(n), wf: n.Val}, nil
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
