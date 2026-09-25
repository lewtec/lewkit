// Package wallpaper sets a static desktop background.
//
//	err := wallpaper.SetStatic(ctx, path)
//
// Import swaybg or feh. Swaybg is Wayland. Feh is X11.
// Both start the tool with systemd-run --user.
package wallpaper

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver sets one still image as the background.
type Driver interface {
	SetStatic(ctx context.Context, path string) error
}

// SetStatic shows the image at path.
func SetStatic(ctx context.Context, path string) error {
	return driver.With(ctx, func(source Driver) error {
		return source.SetStatic(ctx, path)
	})
}
