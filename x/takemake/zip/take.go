package zip

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
)

func readArchive(raw []byte) (*File, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, ErrFormat
	}
	file := New()
	for _, member := range zr.File {
		entry, err := entryFromHeader(member)
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}
		file.Entries = append(file.Entries, entry)
	}
	return file, nil
}

func entryFromHeader(member *zip.File) (*Entry, error) {
	name, err := cleanName(member.Name)
	if err != nil {
		return nil, err
	}
	if name == "." {
		return nil, nil
	}
	mode := member.Mode()
	entry := &Entry{
		Name: name,
		Time: wall(member.Modified),
	}
	dir := mode&fs.ModeDir != 0
	link := mode&fs.ModeSymlink != 0
	if dir && link {
		return nil, ErrKind
	}
	if dir {
		entry.Mode = fs.ModeDir | permission(mode)
		return entry, nil
	}
	rc, err := member.Open()
	if err != nil {
		if errors.Is(err, zip.ErrAlgorithm) {
			return nil, ErrCompress
		}
		return nil, ErrFormat
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, ErrFormat
	}
	if link {
		target := string(data)
		if target == "" || strings.ContainsRune(target, 0) {
			return nil, ErrName
		}
		perm := permission(mode)
		if perm == 0 {
			perm = linkMode
		}
		entry.Mode = fs.ModeSymlink | perm
		entry.Target = target
		return entry, nil
	}
	entry.Mode = permission(mode)
	if len(data) > 0 {
		entry.Data = data
	}
	return entry, nil
}
