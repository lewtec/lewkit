// Package png encodes and decodes PNG bytes.
package png

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/lewtec/lewkit/x/image/convert"
)

// Match reports whether raw starts with a PNG signature.
func Match(raw []byte) bool {
	return bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G'})
}

// Decode reads a PNG image.
func Decode(raw []byte) (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: png: %w", convert.ErrFormat, err)
	}
	return img, nil
}

// Encode writes img as PNG bytes.
func Encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
