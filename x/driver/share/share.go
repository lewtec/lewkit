// Package share offers text, a URL, or files to another app.
//
//	err := share.Out(ctx, share.Item{Text: "hello"})
//
// The desktop backend copies text to the clipboard, or opens the first file.
package share

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

// ErrEmptyItem means Text, URL, and Paths are all empty.
var ErrEmptyItem = errors.New("share item is empty")

// Item is one outbound share. At least one of Text, URL, or Paths must be set.
type Item struct {
	Title string
	Text  string
	URL   string
	Paths []string
}

// Driver presents Item to the host.
type Driver interface {
	Out(ctx context.Context, item Item) error
}

// Out presents item with the selected backend.
func Out(ctx context.Context, item Item) error {
	if err := item.validate(); err != nil {
		return err
	}
	return driver.With(ctx, func(d Driver) error {
		return d.Out(ctx, item)
	})
}

func (item Item) validate() error {
	if strings.TrimSpace(item.Text) == "" && strings.TrimSpace(item.URL) == "" && len(item.Paths) == 0 {
		return ErrEmptyItem
	}
	for _, p := range item.Paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("share path must be absolute: %q", p)
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("share path: %w", err)
		}
	}
	return nil
}
