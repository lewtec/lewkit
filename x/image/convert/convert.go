// Package convert pads, resizes, and squares an image.
//
// PNG, JPEG, ICO, and ICNS bytes live in their own packages. file reads
// those four when the header names the format. An SVG document lives in
// the svg package, and svg/render paints that document into an image.
package convert

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// ErrFormat means the image cannot be squared, or the bytes are not a format
// the decoder accepted.
var ErrFormat = errors.New("image format")

// LinuxSizes are the Status Notifier pixmap sizes hosts pick from.
var LinuxSizes = []int{16, 22, 24, 32, 48}

// Pad centers src on a transparent square. The side is the longer edge.
func Pad(src image.Image) *image.NRGBA {
	return padSquare(src)
}

// Resize scales src to size×size with CatmullRom.
func Resize(src image.Image, size int) *image.NRGBA {
	return resize(src, size)
}

// Square pads src to a square and resizes it to size×size.
func Square(src image.Image, size int) (*image.NRGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%w: nil image", ErrFormat)
	}
	if size <= 0 {
		return nil, fmt.Errorf("%w: size %d", ErrFormat, size)
	}
	return resize(padSquare(src), size), nil
}

// Squares pads src once and returns one square image per size.
func Squares(src image.Image, sizes []int) ([]*image.NRGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%w: nil image", ErrFormat)
	}
	square := padSquare(src)
	out := make([]*image.NRGBA, 0, len(sizes))
	for _, size := range sizes {
		if size <= 0 {
			return nil, fmt.Errorf("%w: size %d", ErrFormat, size)
		}
		out = append(out, resize(square, size))
	}
	return out, nil
}

// ARGB encodes img as ARGB32, network byte order, straight alpha.
func ARGB(img image.Image) []byte {
	if img == nil {
		return nil
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	out := make([]byte, 0, width*height*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a, r, g, b := straight(img.At(x, y))
			out = append(out, a, r, g, b)
		}
	}
	return out
}

func padSquare(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	side := width
	if height > side {
		side = height
	}
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	offset := image.Pt((side-width)/2, (side-height)/2)
	rect := image.Rect(offset.X, offset.Y, offset.X+width, offset.Y+height)
	draw.Draw(dst, rect, src, bounds.Min, draw.Over)
	return dst
}

func resize(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func straight(c color.Color) (a, r, g, b byte) {
	r16, g16, b16, a16 := c.RGBA()
	a = byte(a16 >> 8)
	if a16 == 0 {
		return 0, 0, 0, 0
	}
	r = byte((r16 * 65535 / a16) >> 8)
	g = byte((g16 * 65535 / a16) >> 8)
	b = byte((b16 * 65535 / a16) >> 8)
	return a, r, g, b
}
