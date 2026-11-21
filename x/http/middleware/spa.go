// Package middleware holds HTTP middleware.
//
// [SPA] serves an [io/fs.FS] with the goftpd SPA rules.
// A real file wins. A directory serves index.html when that file
// is in the directory. A miss looks only at the filesystem root:
// 404.html with status 404, then index.html with status 200.
// The request then goes to the next handler.
package middleware

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

const cacheControl = "max-age=5"

type spaHandler struct {
	filesystem fs.FS
	next       http.Handler
}

// SPA serves filesystem with the goftpd SPA rules.
//
// GET and HEAD read filesystem. Any other method is passed to next.
// A nil next answers 405.
//
// A path that names a directory and has no final slash is redirected
// with status 308. A miss is not redirected. A file is not redirected.
//
// A real file is served with [http.ServeFileFS] and status 200.
// A directory that contains index.html is that file.
// A directory that does not contain index.html is a miss.
// SPA does not list a directory.
//
// A miss looks only at the filesystem root. It does not walk to a
// parent index.html. 404.html is served with status 404.
// index.html is served with status 200.
// If neither file exists, the request is passed to next.
// A nil next writes status 404.
//
// Responses set Cache-Control to max-age=5.
// The query string is not part of the file name.
//
// Names go through [path.Path]. That type does not record a trailing
// slash, so the redirect check reads the request path.
func SPA(filesystem fs.FS, next http.Handler) http.Handler {
	return spaHandler{filesystem: filesystem, next: next}
}

func (handler spaHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		if handler.next != nil {
			handler.next.ServeHTTP(response, request)
			return
		}
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, slash := requestName(request.URL.Path)
	if !name.Valid() || name.IsAbs() {
		handler.miss(response, request)
		return
	}
	file, err := name.IsFile(handler.filesystem)
	if err != nil {
		handler.statFailed(response, request, name, err)
		return
	}
	if file {
		handler.serve(response, request, name, http.StatusOK)
		return
	}
	directory, err := name.IsDir(handler.filesystem)
	if err != nil {
		handler.statFailed(response, request, name, err)
		return
	}
	if directory && name.String() != "." && !slash {
		target := *request.URL
		target.Path = "/" + name.String() + "/"
		http.Redirect(response, request, target.String(), http.StatusPermanentRedirect)
		return
	}
	if directory {
		indexName := name.Join("index.html")
		indexFile, err := indexName.IsFile(handler.filesystem)
		if err != nil {
			handler.statFailed(response, request, indexName, err)
			return
		}
		if indexFile {
			handler.serve(response, request, indexName, http.StatusOK)
			return
		}
	}
	handler.miss(response, request)
}

func (handler spaHandler) miss(response http.ResponseWriter, request *http.Request) {
	notFoundName := path.New("404.html")
	notFoundFile, err := notFoundName.IsFile(handler.filesystem)
	if err != nil {
		handler.statFailed(response, request, notFoundName, err)
		return
	}
	if notFoundFile {
		handler.serve(response, request, notFoundName, http.StatusNotFound)
		return
	}
	indexName := path.New("index.html")
	indexFile, err := indexName.IsFile(handler.filesystem)
	if err != nil {
		handler.statFailed(response, request, indexName, err)
		return
	}
	if indexFile {
		handler.serve(response, request, indexName, http.StatusOK)
		return
	}
	if handler.next != nil {
		handler.next.ServeHTTP(response, request)
		return
	}
	http.NotFound(response, request)
}

func (handler spaHandler) serve(response http.ResponseWriter, request *http.Request, name path.Path, status int) {
	response.Header().Set("Cache-Control", cacheControl)
	if status != http.StatusOK {
		response = &statusResponse{ResponseWriter: response, status: status}
	}
	http.ServeFileFS(response, request, handler.filesystem, name.String())
}

func (handler spaHandler) statFailed(response http.ResponseWriter, request *http.Request, name path.Path, err error) {
	slog.WarnContext(request.Context(), "stat", "path", name.String(), "err", err)
	http.Error(response, "stat file", http.StatusInternalServerError)
}

// requestName maps a URL path to an [io/fs] name.
// The bool is true when the URL path ends in a slash.
func requestName(urlPath string) (path.Path, bool) {
	if urlPath == "" {
		urlPath = "/"
	}
	return path.New(strings.Trim(urlPath, "/")), strings.HasSuffix(urlPath, "/")
}

// statusResponse keeps a non-200 status when [http.ServeFileFS] writes 200.
type statusResponse struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (writer *statusResponse) WriteHeader(code int) {
	if writer.wrote {
		return
	}
	writer.wrote = true
	if code == http.StatusOK {
		code = writer.status
	}
	writer.ResponseWriter.WriteHeader(code)
}

func (writer *statusResponse) Write(body []byte) (int, error) {
	writer.WriteHeader(http.StatusOK)
	return writer.ResponseWriter.Write(body)
}
