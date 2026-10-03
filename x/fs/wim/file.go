//go:build linux || windows

package wim

import (
	"cmp"
	"io"
	"io/fs"
	"slices"
	"time"

	winwim "github.com/Microsoft/go-winio/wim"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

type wimNode struct {
	f *winwim.File
}

func (n wimNode) Name() string {
	if n.f == nil {
		return ""
	}
	return n.f.Name
}

func (n wimNode) IsDir() bool {
	return n.f != nil && n.f.IsDir()
}

func (n wimNode) info() fs.FileInfo {
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	name := n.Name()
	if n.IsDir() {
		mode = fs.ModeDir | 0o555
	}
	if n.f != nil {
		size = n.f.Size
		mod = n.f.LastWriteTime.Time()
	}
	return lewfs.FileInfo(name, size, mode, mod)
}

func (n wimNode) open() (fs.File, error) {
	if n.f == nil || n.IsDir() {
		return nil, &fs.PathError{Op: "open", Path: n.Name(), Err: fs.ErrInvalid}
	}
	return &file{info: n.info(), wf: n.f}, nil
}

func (n wimNode) children() ([]entry, error) {
	if n.f == nil {
		return nil, fs.ErrInvalid
	}
	kids, err := n.f.Readdir()
	if err != nil {
		return nil, err
	}
	out := make([]entry, 0, len(kids))
	for _, k := range kids {
		if k == nil {
			continue
		}
		name := k.Name
		if name == "" || name == "." || name == ".." {
			continue
		}
		out = append(out, wimNode{f: k})
	}
	return out, nil
}

func listEntries(n entry) ([]fs.DirEntry, error) {
	kids, err := n.children()
	if err != nil {
		return nil, err
	}
	out := make([]fs.DirEntry, 0, len(kids))
	for _, k := range kids {
		out = append(out, fs.FileInfoToDirEntry(k.info()))
	}
	slices.SortFunc(out, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})
	return out, nil
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
