// Package asset serves vendored browser libraries under /__lewkit__/.
//
// Blank-import [github.com/lewtec/lewkit/x/http/asset/prelude] to register
// htmx, tailwindcss, jquery, and sakuracss. [Mount] serves the files that
// those packages registered. A request outside [Prefix] goes to the next
// handler.
package asset

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

const cacheControl = "max-age=5"

// Prefix is the URL directory for registered files.
const Prefix = "/__lewkit__/"

// ErrName is [Register] with an empty name or a name that is not one path segment.
var ErrName = errors.New("invalid asset name")

// ErrExist is [Register] when the name is already in the registry.
var ErrExist = errors.New("already registered")

// ErrEmpty is [Register] with an empty body.
var ErrEmpty = errors.New("empty asset")

// File is one browser library served at [Prefix] plus [File.Name].
type File struct {
	Name        string
	ContentType string
	Body        []byte
}

type registry struct {
	files []File
}

var std = &registry{}

// Register adds one file to the process-wide registry.
// Call it from init, before [Mount] serves requests.
func Register(file File) error {
	return std.register(file)
}

// MustRegister is [Register] that panics on error.
func MustRegister(file File) {
	if err := Register(file); err != nil {
		panic(err)
	}
}

func (set *registry) register(file File) error {
	if file.Name == "" || file.Name == "." || file.Name == ".." || strings.Contains(file.Name, "/") {
		return fmt.Errorf("%s: %w", file.Name, ErrName)
	}
	if len(file.Body) == 0 {
		return fmt.Errorf("%s: %w", file.Name, ErrEmpty)
	}
	for _, existing := range set.files {
		if existing.Name == file.Name {
			return fmt.Errorf("%s: %w", file.Name, ErrExist)
		}
	}
	file.Body = append([]byte(nil), file.Body...)
	set.files = append(set.files, file)
	return nil
}

func (set *registry) find(name string) (File, bool) {
	for _, file := range set.files {
		if file.Name == name {
			return file, true
		}
	}
	return File{}, false
}

// Mount serves registered files under [Prefix].
//
// GET and HEAD read the file. Any other method on a prefix path is 405.
// A prefix path that is not registered is 404.
// A path outside [Prefix] goes to next. A nil next answers 404.
//
// Responses set Cache-Control to max-age=5.
func Mount(next http.Handler) http.Handler {
	return assetHandler{next: next}
}

type assetHandler struct {
	next http.Handler
}

func (handler assetHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	name, ok := assetName(request.URL.Path)
	if !ok {
		if handler.next != nil {
			handler.next.ServeHTTP(response, request)
			return
		}
		http.NotFound(response, request)
		return
	}
	file, ok := std.find(name)
	if !ok {
		http.NotFound(response, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	response.Header().Set("Content-Type", file.ContentType)
	response.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(response, request, name, time.Time{}, bytes.NewReader(file.Body))
}

// assetName returns the file name when urlPath is under [Prefix].
func assetName(urlPath string) (string, bool) {
	if urlPath == "" {
		urlPath = "/"
	}
	cleaned := path.Clean(urlPath)
	if !strings.HasPrefix(cleaned, Prefix) {
		return "", false
	}
	name := strings.TrimPrefix(cleaned, Prefix)
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}
