// Package middleware holds HTTP middleware.
//
// [SPA] serves an [io/fs.FS] with the goftpd SPA rules.
// A real file wins. A directory serves index.html when that file
// is in the directory. A miss looks only at the filesystem root:
// 404.html with status 404, then index.html with status 200.
// The request then goes to the next handler.
package middleware

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	stdpath "path"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

const cacheControl = "max-age=5"

type spaCall struct {
	filesystem fs.FS
	next       http.Handler
	response   http.ResponseWriter
	request    *http.Request
}

// SPA serves filesystem with the goftpd SPA rules.
//
// GET and HEAD read filesystem. Any other method is passed to next.
// A nil next answers 405.
//
// A path that names a directory and has no final slash is redirected
// with status 308. A miss is not redirected. A file is not redirected.
//
// A real file is served with [http.ServeContent] and status 200.
// A directory that contains index.html is that file.
// A directory that does not contain index.html is a miss.
// SPA does not list a directory.
//
// A miss looks only at the filesystem root. It does not walk to a
// parent index.html. 404.html is served with status 404 and is not
// passed to [http.ServeContent]. index.html is served with status 200.
// If neither file exists, the request is passed to next.
// A nil next writes status 404.
//
// Responses set Cache-Control to max-age=5.
// The query string is not part of the file name.
func SPA(filesystem fs.FS, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		call := spaCall{
			filesystem: filesystem,
			next:       next,
			response:   response,
			request:    request,
		}
		call.serve()
	})
}

func (call spaCall) serve() {
	if call.request.Method != http.MethodGet && call.request.Method != http.MethodHead {
		if call.next != nil {
			call.next.ServeHTTP(call.response, call.request)
			return
		}
		call.response.Header().Set("Allow", "GET, HEAD")
		http.Error(call.response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlPath := cleanURLPath(call.request.URL.Path)
	name, ok := filesystemName(urlPath)
	if !ok {
		call.serveMiss()
		return
	}
	info, err := name.Stat(call.filesystem)
	if err != nil {
		call.serveMiss()
		return
	}
	if info.IsDir() {
		if !strings.HasSuffix(urlPath, "/") {
			target := *call.request.URL
			target.Path = urlPath + "/"
			http.Redirect(call.response, call.request, target.String(), http.StatusPermanentRedirect)
			return
		}
		indexName := name.Join("index.html")
		if fileExists(call.filesystem, indexName) {
			call.serveFile(indexName, http.StatusOK)
			return
		}
		call.serveMiss()
		return
	}
	call.serveFile(name, http.StatusOK)
}

func (call spaCall) serveMiss() {
	notFoundName := path.New("404.html")
	if fileExists(call.filesystem, notFoundName) {
		call.serveFile(notFoundName, http.StatusNotFound)
		return
	}
	indexName := path.New("index.html")
	if fileExists(call.filesystem, indexName) {
		call.serveFile(indexName, http.StatusOK)
		return
	}
	if call.next != nil {
		call.next.ServeHTTP(call.response, call.request)
		return
	}
	http.NotFound(call.response, call.request)
}

func (call spaCall) serveFile(name path.Path, status int) {
	file, err := name.Open(call.filesystem)
	if err != nil {
		http.NotFound(call.response, call.request)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(call.response, call.request)
		return
	}
	reader, ok := file.(io.ReadSeeker)
	if !ok {
		http.NotFound(call.response, call.request)
		return
	}
	call.response.Header().Set("Cache-Control", cacheControl)
	if status == http.StatusOK {
		http.ServeContent(call.response, call.request, info.Name(), info.ModTime(), reader)
		return
	}
	contentType := mime.TypeByExtension(name.Suffix())
	if contentType == "" {
		buffer := make([]byte, 512)
		count, err := io.ReadFull(file, buffer)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			http.NotFound(call.response, call.request)
			return
		}
		contentType = http.DetectContentType(buffer[:count])
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			http.NotFound(call.response, call.request)
			return
		}
	}
	call.response.Header().Set("Content-Type", contentType)
	call.response.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	call.response.WriteHeader(status)
	if call.request.Method == http.MethodHead {
		return
	}
	if _, err := io.Copy(call.response, reader); err != nil {
		slog.WarnContext(call.request.Context(), "copy file", "path", name.String(), "err", err)
	}
}

func fileExists(filesystem fs.FS, name path.Path) bool {
	info, err := name.Stat(filesystem)
	return err == nil && !info.IsDir()
}

func cleanURLPath(urlPath string) string {
	if urlPath == "" {
		return "/"
	}
	cleaned := stdpath.Clean(urlPath)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if cleaned != "/" && strings.HasSuffix(urlPath, "/") {
		cleaned += "/"
	}
	return cleaned
}

func filesystemName(urlPath string) (path.Path, bool) {
	trimmed := strings.TrimSuffix(urlPath, "/")
	if trimmed == "" || trimmed == "/" {
		return path.New(), true
	}
	name := path.New(strings.TrimPrefix(trimmed, "/"))
	if !name.Valid() || name.IsAbs() {
		return path.Path{}, false
	}
	return name, true
}
