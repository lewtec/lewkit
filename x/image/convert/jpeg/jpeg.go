// Package jpeg encodes and decodes JPEG bytes.
package jpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/lewtec/lewkit/x/image/convert"
)

// Match reports whether raw starts with a JPEG start-of-image marker.
func Match(raw []byte) bool {
	return bytes.HasPrefix(raw, []byte{0xff, 0xd8})
}

// Decode reads a JPEG image.
func Decode(raw []byte) (image.Image, error) {
	img, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: jpeg: %w", convert.ErrFormat, err)
	}
	return img, nil
}

// Encode writes img as JPEG bytes at quality 90.
func Encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
