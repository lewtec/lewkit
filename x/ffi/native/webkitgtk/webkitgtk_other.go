//go:build !linux || android

package webkitgtk

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnavailable means the WebKitGTK libraries are not loaded.
var ErrUnavailable = errors.New("webkitgtk unavailable")

// Symbols is the loaded C API. It is never returned on this OS.
type Symbols struct{}

// Load reports that WebKitGTK is a Linux library.
func Load(ctx context.Context) (*Symbols, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: not linux", ErrUnavailable)
}
