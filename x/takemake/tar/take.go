package tar

import (
	"archive/tar"
	"errors"
	"io"
	"io/fs"
	"strings"
)

func readArchive(r io.Reader) (*File, error) {
	tr := tar.NewReader(r)
	file := New()
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return file, nil
		}
		if err != nil || hdr == nil {
			return nil, ErrFormat
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		entry, err := entryFromHeader(tr, hdr)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}
		file.Entries = append(file.Entries, entry)
	}
}

func entryFromHeader(tr *tar.Reader, hdr *tar.Header) (*Entry, error) {
	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeRegA, tar.TypeDir, tar.TypeSymlink, tar.TypeGNUSparse:
	default:
		return nil, ErrEntry
	}
	if hdr.Size < 0 {
		return nil, ErrFormat
	}
	name, err := cleanName(hdr.Name)
	if err != nil {
		return nil, err
	}
	if name == "." {
		return nil, nil
	}
	mode := hdr.FileInfo().Mode()
	entry := &Entry{
		Name: name,
		Time: hdr.ModTime.UTC(),
	}
	switch {
	case hdr.Typeflag == tar.TypeDir || mode&fs.ModeDir != 0:
		entry.Mode = fs.ModeDir | permission(mode)
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return nil, ErrFormat
		}
		return entry, nil
	case hdr.Typeflag == tar.TypeSymlink || mode&fs.ModeSymlink != 0:
		if hdr.Linkname == "" || strings.ContainsRune(hdr.Linkname, 0) {
			return nil, ErrName
		}
		entry.Mode = fs.ModeSymlink | permission(mode)
		entry.Target = hdr.Linkname
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return nil, ErrFormat
		}
		return entry, nil
	default:
		data, err := io.ReadAll(tr)
		if err != nil || int64(len(data)) != hdr.Size {
			return nil, ErrFormat
		}
		entry.Mode = permission(mode)
		if len(data) > 0 {
			entry.Data = data
		}
		return entry, nil
	}
}
