// Package site serves a static site with Range requests.
// A zip whose only top directory is the site root is served from that directory.
// Files that cannot seek are buffered so audio and video Range requests work.
package site

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/zip"
	"github.com/lewtec/lewkit/x/http/middleware"
)

// Zip serves a zip archive. r must be an [io.ReaderAt] with a known size.
// ctx ends the open. The handler does not use ctx after Zip returns.
func Zip(ctx context.Context, r io.Reader) (http.Handler, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	archive, err := zip.Open(ctx, r)
	if err != nil {
		return nil, err
	}
	files, err := lewfs.StripTopDirectory(archive)
	if err != nil {
		return nil, err
	}
	return Dir(files), nil
}

// Dir serves filesystem with the SPA rules in [middleware.SPA].
func Dir(filesystem fs.FS) http.Handler {
	return middleware.SPA(seekFS{FS: filesystem}, nil)
}

// seekFS buffers files that cannot seek.
// Directories and files that already seek are returned as stored.
type seekFS struct {
	fs.FS
}

func (root seekFS) Open(name string) (fs.File, error) {
	file, err := root.FS.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.IsDir() {
		return file, nil
	}
	if _, ok := file.(io.Seeker); ok {
		return file, nil
	}
	data, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return &memFile{Reader: bytes.NewReader(data), info: info}, nil
}

type memFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (file *memFile) Stat() (fs.FileInfo, error) { return file.info, nil }

func (file *memFile) Close() error { return nil }
