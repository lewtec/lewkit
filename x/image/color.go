package image

import (
	"fmt"
	stdimage "image"
	"strings"
)

// Order is how the channels are packed into bytes. The zero value is [RGB].
type Order uint8

const (
	// RGB packs red, green, blue, then alpha.
	RGB Order = iota
	// BGR packs blue, green, red, then alpha.
	BGR
)

// Color is one straight color, 0..255. Red, Green, and Blue are the logical
// channels. Order says how [Color.Bytes] lays those channels out.
type Color struct {
	Red, Green, Blue, Alpha uint8
	Order                   Order
}

// Bytes packs the color in its Order.
func (c Color) Bytes() [4]byte {
	if c.Order == BGR {
		return [4]byte{c.Blue, c.Green, c.Red, c.Alpha}
	}
	return [4]byte{c.Red, c.Green, c.Blue, c.Alpha}
}

// WithOrder returns c marked as order. The logical channels stay put.
func (c Color) WithOrder(order Order) Color {
	c.Order = order
	return c
}

// Unpack reads four packed bytes. order says which byte is red.
func Unpack(packed [4]byte, order Order) Color {
	if order == BGR {
		return Color{Red: packed[2], Green: packed[1], Blue: packed[0], Alpha: packed[3], Order: order}
	}
	return Color{Red: packed[0], Green: packed[1], Blue: packed[2], Alpha: packed[3], Order: order}
}

// ParseColor reads #RGB, #RRGGBB, or #RRGGBBAA. The leading # is optional.
func ParseColor(text string) (Color, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "#")
	if !hex(text) {
		return Color{}, fmt.Errorf("color %q", text)
	}
	var r, g, b, a uint8
	a = 255
	switch len(text) {
	case 3:
		r, g, b = hexVal(text[0])*17, hexVal(text[1])*17, hexVal(text[2])*17
	case 6:
		r, g, b = byte2(text[0:2]), byte2(text[2:4]), byte2(text[4:6])
	case 8:
		r, g, b, a = byte2(text[0:2]), byte2(text[2:4]), byte2(text[4:6]), byte2(text[6:8])
	default:
		return Color{}, fmt.Errorf("color %q", text)
	}
	return Color{Red: r, Green: g, Blue: b, Alpha: a, Order: RGB}, nil
}

// Average is the alpha-weighted mean of opaque pixels. A blank image is black.
func Average(img stdimage.Image) Color {
	if img == nil {
		return Color{Alpha: 255, Order: RGB}
	}
	bounds := img.Bounds()
	var red, green, blue, weight uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 16*256 {
				continue
			}
			red += uint64(r) * uint64(a)
			green += uint64(g) * uint64(a)
			blue += uint64(b) * uint64(a)
			weight += uint64(a)
		}
	}
	if weight == 0 {
		return Color{Alpha: 255, Order: RGB}
	}
	return Color{
		Red:   uint8((red / weight) >> 8),
		Green: uint8((green / weight) >> 8),
		Blue:  uint8((blue / weight) >> 8),
		Alpha: 255,
		Order: RGB,
	}
}

func byte2(s string) uint8 {
	var n uint8
	for i := 0; i < len(s); i++ {
		n = n<<4 | hexVal(s[i])
	}
	return n
}

func hex(s string) bool {
	for i := 0; i < len(s); i++ {
		if hexVal(s[i]) == 0xff {
			return false
		}
	}
	return true
}

func hexVal(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0xff
	}
}
