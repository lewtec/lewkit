// Package camera lists video devices and captures one still frame.
//
//	cams, err := camera.List(ctx)
//	img, err := cams[0].Capture(ctx)
//
// Import the linux implementation. It enumerates /dev/video* and captures
// one PNG frame with ffmpeg.
package camera

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver discovers cameras.
type Driver interface {
	// List returns the cameras available now.
	List(ctx context.Context) ([]Camera, error)
}

// Camera is one capture device.
type Camera interface {
	// ID is a stable device identifier.
	ID() string
	// Name is a human-readable label.
	Name() string
	// Capture takes one still frame.
	Capture(ctx context.Context) (image.Image, error)
}

// List returns the cameras available now.
func List(ctx context.Context) ([]Camera, error) {
	return driver.WithResult(ctx, func(source Driver) ([]Camera, error) {
		return source.List(ctx)
	})
}
