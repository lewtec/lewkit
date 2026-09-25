// Package clipboard writes text and images to the host clipboard.
//
//	err := clipboard.WriteText(ctx, "hello")
//
// Backends are wl-copy, xclip, pbcopy, and the Windows clip command.
package clipboard

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver writes one payload to the clipboard.
type Driver interface {
	WriteText(ctx context.Context, text string) error
	WriteImage(ctx context.Context, img image.Image) error
}

// WriteText copies text.
func WriteText(ctx context.Context, text string) error {
	return driver.With(ctx, func(d Driver) error {
		return d.WriteText(ctx, text)
	})
}

// WriteImage copies img as a PNG.
func WriteImage(ctx context.Context, img image.Image) error {
	return driver.With(ctx, func(d Driver) error {
		return d.WriteImage(ctx, img)
	})
}
