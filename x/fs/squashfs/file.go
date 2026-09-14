package squashfs

import (
	"io"
	"io/fs"
)

type file struct {
	fs.File
}

func (f *file) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return f.File.Read(p)
}

type dirFile struct {
	info fs.FileInfo
	ents []fs.DirEntry
	off  int
}

func (d *dirFile) Stat() (fs.FileInfo, error) { return d.info, nil }

func (d *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.Name(), Err: fs.ErrInvalid}
}

func (d *dirFile) Close() error { return nil }

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.off >= len(d.ents) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := d.ents[d.off:]
		d.off = len(d.ents)
		return out, nil
	}
	end := d.off + n
	if end > len(d.ents) {
		end = len(d.ents)
	}
	out := d.ents[d.off:end]
	d.off = end
	return out, nil
}
