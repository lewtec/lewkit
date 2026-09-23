package tray

import (
	"fmt"
	"image"

	"github.com/lewtec/lewkit/x/image/convert"
)

// Source returns the icon pixels. A nil image and a nil error means Name only.
func Source(icon Icon) (image.Image, error) {
	if icon.Image != nil {
		return icon.Image, nil
	}
	if len(icon.Bytes) == 0 {
		return nil, nil
	}
	img, err := convert.Decode(icon.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIcon, err)
	}
	return img, nil
}
