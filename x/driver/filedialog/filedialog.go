// Package filedialog asks the user for files or folders.
//
//	paths, err := filedialog.Choose(ctx, filedialog.Request{
//		Title:   "Open",
//		Filters: []filedialog.Filter{{Name: "Audio", Patterns: []string{"*.mp3", "*.flac"}}},
//	})
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (gtk, qt, cocoa, win32).
//
// Linux GTK uses the gtk portal file chooser. Linux Qt uses the KDE portal
// file chooser. Windows uses the common item dialog. macOS uses NSOpenPanel
// and NSSavePanel. On macOS and Windows, call
// [github.com/lewtec/lewkit/x/thread.Bind] from main and run
// [github.com/lewtec/lewkit/x/thread.Loop]. [github.com/lewtec/lewkit/x/thread.Run]
// does both. Choose may be called from another goroutine while Loop is running.
//
// Patterns are globs such as "*.mp3". Folder chooses directories. Save chooses
// one new path. Multiple chooses more than one path. Save with Folder or
// Multiple is rejected.
package filedialog

import (
	"context"
	"errors"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	// ErrCanceled means the user dismissed the dialog.
	ErrCanceled = errors.New("file dialog canceled")
	// ErrRequest means the request cannot be shown.
	ErrRequest = errors.New("file dialog request")

	errSaveFolder = errors.New("save a folder")
	errSaveMany   = errors.New("save more than one file")
)

// Filter limits the files a dialog lists. Patterns are globs.
type Filter struct {
	Name     string
	Patterns []string
}

// Request is one dialog.
// Directory is the folder the dialog opens in.
// Name is the suggested file name for Save.
type Request struct {
	Title     string
	Directory string
	Name      string
	Filters   []Filter
	Multiple  bool
	Folder    bool
	Save      bool
}

// Validate reports a request no driver can show.
func (r Request) Validate() error {
	if r.Save && r.Folder {
		return errSaveFolder
	}
	if r.Save && r.Multiple {
		return errSaveMany
	}
	return nil
}

// Extensions returns file extensions from glob patterns such as "*.mp3".
// A pattern that is not a single extension is omitted.
func Extensions(filters []Filter) []string {
	var out []string
	for _, item := range filters {
		for _, pattern := range item.Patterns {
			ext := strings.TrimPrefix(pattern, "*.")
			ext = strings.TrimPrefix(ext, ".")
			if ext == "" || strings.ContainsAny(ext, "*?/") {
				continue
			}
			out = append(out, ext)
		}
	}
	return out
}

// TitleOrDefault returns the dialog title, or a short default.
func (r Request) TitleOrDefault() string {
	if r.Title != "" {
		return r.Title
	}
	if r.Save {
		return "Save"
	}
	if r.Folder {
		return "Choose folder"
	}
	return "Open"
}

// Driver shows a file or folder dialog.
type Driver interface {
	Choose(ctx context.Context, req Request) ([]string, error)
}

// Choose asks the active file dialog driver for paths.
// A dismiss is [ErrCanceled]. A bad request is [ErrRequest].
func Choose(ctx context.Context, req Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmtRequest(err)
	}
	return driver.WithResult(ctx, func(d Driver) ([]string, error) {
		return d.Choose(ctx, req)
	})
}

func fmtRequest(err error) error {
	return errors.Join(ErrRequest, err)
}
