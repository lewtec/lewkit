package window

import (
	"image"
	"image/color"
	"sync"

	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/image/convert"
)

// ShowShell makes this process a foreground app and sets its switcher icon.
// title names the menu when the bundle has no display name.
// A nil icon uses the built-in mark. Hosts with no shell icon ignore the call.
// On darwin, call it on the UI thread.
func ShowShell(title string, icon image.Image) {
	showShell(title, shellImage(icon))
}

// ShellPicture is the _NET_WM_ICON property: for each size, width, height,
// then straight-alpha ARGB with alpha in the high byte.
// A nil icon is the built-in mark at 32, 128, and 256.
func ShellPicture(icon image.Image) []uint32 {
	words := shellWords(icon)
	if len(words) == 0 {
		return nil
	}
	out := make([]uint32, len(words))
	copy(out, words)
	return out
}

func shellImage(icon image.Image) image.Image {
	if icon == nil {
		return defaultShellMark()
	}
	return icon
}

var (
	markOnce sync.Once
	markImg  image.Image
)

func defaultShellMark() image.Image {
	markOnce.Do(func() {
		img, err := icons.DefaultMark()
		if err != nil {
			return
		}
		markImg = img
	})
	return markImg
}

var (
	wordsOnce sync.Once
	words     []uint32
)

func shellWords(icon image.Image) []uint32 {
	if icon != nil {
		return packImage(icon)
	}
	wordsOnce.Do(func() {
		src := defaultShellMark()
		if src == nil {
			return
		}
		tiles, err := convert.Squares(src, []int{32, 128, 256})
		if err != nil {
			return
		}
		for _, tile := range tiles {
			words = append(words, packImage(tile)...)
		}
	})
	return words
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
