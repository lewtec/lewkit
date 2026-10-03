package window

import (
	"image"
	"image/color"
)

// ShowShell makes this process a foreground app and sets its switcher icon.
// title names the menu when the bundle has no display name.
// A nil icon leaves the host icon unchanged. Hosts with no shell icon ignore the call.
// On darwin, call it on the UI thread.
func ShowShell(title string, icon image.Image) {
	showShell(title, icon)
}

// ShellPicture is the _NET_WM_ICON property: width, height, then
// straight-alpha ARGB with alpha in the high byte.
// A nil icon is an empty property.
func ShellPicture(icon image.Image) []uint32 {
	words := packImage(icon)
	if len(words) == 0 {
		return nil
	}
	out := make([]uint32, len(words))
	copy(out, words)
	return out
}

func packImage(img image.Image) []uint32 {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil
	}
	out := make([]uint32, 2+w*h)
	out[0] = uint32(w)
	out[1] = uint32(h)
	i := 2
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out[i] = argb(img.At(x, y))
			i++
		}
	}
	return out
}

// argb unpremultiplies like convert.straight. color.Color.RGBA is premultiplied.
func argb(c color.Color) uint32 {
	r16, g16, b16, a16 := c.RGBA()
	if a16 == 0 {
		return 0
	}
	r := byte((r16 * 65535 / a16) >> 8)
	g := byte((g16 * 65535 / a16) >> 8)
	b := byte((b16 * 65535 / a16) >> 8)
	a := byte(a16 >> 8)
	return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}
