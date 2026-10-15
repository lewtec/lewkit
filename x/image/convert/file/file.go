// Package file reads PNG, JPEG, ICO, and ICNS bytes into an image.
// The format is chosen from the header.
package file

import (
	"bytes"
	"fmt"
	"image"
	"os"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/lewtec/lewkit/x/image/convert/icns"
	"github.com/lewtec/lewkit/x/image/convert/ico"
	"github.com/lewtec/lewkit/x/image/convert/jpeg"
	"github.com/lewtec/lewkit/x/image/convert/png"
)

// Open reads a file Decode can read.
func Open(path string) (image.Image, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(raw)
}

// Decode reads a PNG, JPEG, ICO, or ICNS. ICO and ICNS return the largest image.
func Decode(raw []byte) (image.Image, error) {
	switch {
	case png.Match(raw):
		return png.Decode(raw)
	case jpeg.Match(raw):
		return jpeg.Decode(raw)
	case icns.Match(raw):
		return icns.Decode(raw)
	case ico.Match(raw):
		return ico.Decode(raw)
	default:
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: png, jpeg, ico, or icns", convert.ErrFormat)
		}
		return img, nil
	}
}
