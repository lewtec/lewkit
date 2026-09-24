package image

import (
	"fmt"
	stdimage "image"
	"strings"
)

// Color is straight red, green, blue, and alpha, each 0..255. Paint uses this.
type Color struct {
	Red, Green, Blue, Alpha uint8
}

// BGR is [Color] packed blue, green, red, alpha.
type BGR struct {
	Blue, Green, Red, Alpha uint8
}

// CMYK is cyan, magenta, yellow, and black, each 0..255.
type CMYK struct {
	Cyan, Magenta, Yellow, Black uint8
}

// HSV is a hue in degrees, 0..360, plus saturation and value, 0..255.
type HSV struct {
	Hue        uint16
	Saturation uint8
	Value      uint8
}

// Bytes packs red, green, blue, alpha.
func (c Color) Bytes() [4]byte {
	return [4]byte{c.Red, c.Green, c.Blue, c.Alpha}
}

// BGR returns the same channels in blue, green, red order.
func (c Color) BGR() BGR {
	return BGR{Blue: c.Blue, Green: c.Green, Red: c.Red, Alpha: c.Alpha}
}

// Bytes packs blue, green, red, alpha.
func (c BGR) Bytes() [4]byte {
	return [4]byte{c.Blue, c.Green, c.Red, c.Alpha}
}

// Color returns the logical red, green, and blue.
func (c BGR) Color() Color {
	return Color{Red: c.Red, Green: c.Green, Blue: c.Blue, Alpha: c.Alpha}
}

// CMYK converts the color. Black is what is left after the strongest channel.
func (c Color) CMYK() CMYK {
	r, g, b := int(c.Red), int(c.Green), int(c.Blue)
	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	black := 255 - max
	if black == 255 {
		return CMYK{Black: 255}
	}
	scale := 255 - black
	return CMYK{
		Cyan:    uint8((255 - r - black) * 255 / scale),
		Magenta: uint8((255 - g - black) * 255 / scale),
		Yellow:  uint8((255 - b - black) * 255 / scale),
		Black:   uint8(black),
	}
}

// Color returns an opaque red, green, and blue.
func (c CMYK) Color() Color {
	black := int(c.Black)
	if black == 255 {
		return Color{Alpha: 255}
	}
	scale := 255 - black
	channel := func(v uint8) uint8 {
		return uint8(255 - black - int(v)*scale/255)
	}
	return Color{channel(c.Cyan), channel(c.Magenta), channel(c.Yellow), 255}
}

// HSV converts the color. Hue is 0 for red, 120 for green, and 240 for blue.
func (c Color) HSV() HSV {
	r, g, b := int(c.Red), int(c.Green), int(c.Blue)
	max, min := r, r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}
	if max == 0 {
		return HSV{}
	}
	value := uint8(max)
	if max == min {
		return HSV{Value: value}
	}
	delta := max - min
	sat := uint8(delta * 255 / max)
	var hue int
	switch max {
	case r:
		hue = 60 * (g - b) / delta
	case g:
		hue = 60*(b-r)/delta + 120
	default:
		hue = 60*(r-g)/delta + 240
	}
	if hue < 0 {
		hue += 360
	}
	return HSV{Hue: uint16(hue), Saturation: sat, Value: value}
}

// Color returns an opaque red, green, and blue.
func (c HSV) Color() Color {
	if c.Saturation == 0 {
		return Color{c.Value, c.Value, c.Value, 255}
	}
	hue := int(c.Hue % 360)
	region := hue / 60
	rest := hue % 60
	v := int(c.Value)
	s := int(c.Saturation)
	p := v * (255 - s) / 255
	q := v * (255 - s*rest/60) / 255
	t := v * (255 - s*(60-rest)/60) / 255
	switch region {
	case 0:
		return Color{uint8(v), uint8(t), uint8(p), 255}
	case 1:
		return Color{uint8(q), uint8(v), uint8(p), 255}
	case 2:
		return Color{uint8(p), uint8(v), uint8(t), 255}
	case 3:
		return Color{uint8(p), uint8(q), uint8(v), 255}
	case 4:
		return Color{uint8(t), uint8(p), uint8(v), 255}
	default:
		return Color{uint8(v), uint8(p), uint8(q), 255}
	}
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
	return Color{Red: r, Green: g, Blue: b, Alpha: a}, nil
}

// Average is the alpha-weighted mean of opaque pixels. A blank image is black.
func Average(img stdimage.Image) Color {
	if img == nil {
		return Color{Alpha: 255}
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
		return Color{Alpha: 255}
	}
	return Color{
		Red:   uint8((red / weight) >> 8),
		Green: uint8((green / weight) >> 8),
		Blue:  uint8((blue / weight) >> 8),
		Alpha: 255,
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
