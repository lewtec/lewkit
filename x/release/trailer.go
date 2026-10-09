package release

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sort"
)

const trailerMagic = "LKAI"

var errNoTrailer = errors.New("release: no app trailer")

// AppendTrailer stores files at the end of the executable at path.
// The ELF stays executable. OpenTrailer reads the same bytes back.
func AppendTrailer(path string, files map[string][]byte) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var body bytes.Buffer
	zw := zip.NewWriter(&body)
	for _, name := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			return err
		}
		if _, err := w.Write(files[name]); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(body.Bytes()); err != nil {
		_ = f.Close()
		return err
	}
	var tail [12]byte
	binary.LittleEndian.PutUint64(tail[:8], uint64(body.Len()))
	copy(tail[8:], trailerMagic)
	if _, err := f.Write(tail[:]); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// Trailer is the zip stored after a Linux AppImage executable.
type Trailer struct {
	file *os.File
	zip  *zip.Reader
}

// OpenTrailer opens the trailer of an executable.
// A file with no trailer returns an error.
func OpenTrailer(path string) (*Trailer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if info.Size() < 12 {
		_ = f.Close()
		return nil, errNoTrailer
	}
	var tail [12]byte
	if _, err := f.ReadAt(tail[:], info.Size()-12); err != nil {
		_ = f.Close()
		return nil, err
	}
	if string(tail[8:]) != trailerMagic {
		_ = f.Close()
		return nil, errNoTrailer
	}
	n := int64(binary.LittleEndian.Uint64(tail[:8]))
	start := info.Size() - 12 - n
	if n <= 0 || start < 0 {
		_ = f.Close()
		return nil, errNoTrailer
	}
	r, err := zip.NewReader(io.NewSectionReader(f, start, n), n)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Trailer{file: f, zip: r}, nil
}

// Bytes reads one trailer entry.
func (t *Trailer) Bytes(name string) ([]byte, error) {
	if t == nil || t.zip == nil {
		return nil, errNoTrailer
	}
	for _, file := range t.zip.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		return raw, err
	}
	return nil, os.ErrNotExist
}

// Close releases the executable.
func (t *Trailer) Close() error {
	if t == nil || t.file == nil {
		return nil
	}
	err := t.file.Close()
	t.file = nil
	return err
}
