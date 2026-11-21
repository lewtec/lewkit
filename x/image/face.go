package image

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/singleton"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

const facePoints = 18

var systemFace = singleton.NewSingleton(func(context.Context) (font.Face, error) {
	for _, path := range systemFonts {
		if path == "" {
			continue
		}
		f, err := loadFace(path)
		if err == nil {
			return f, nil
		}
	}
	return basicfont.Face7x13, nil
})

var systemFonts = []string{
	"/System/Library/Fonts/SFNS.ttf",
	"/System/Library/Fonts/SFNSText.ttf",
	"/System/Library/Fonts/Helvetica.ttc",
	"/System/Library/Fonts/Supplemental/Arial.ttf",
	"/Library/Fonts/Arial.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/TTF/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
	"/usr/share/fonts/liberation/LiberationSans-Regular.ttf",
	os.Getenv("WINDIR") + `\Fonts\segoeui.ttf`,
	os.Getenv("WINDIR") + `\Fonts\arial.ttf`,
}

// Face is a system sans-serif face, or 7×13 if none is readable.
func Face() font.Face {
	f, err := singleton.Get(systemFace)
	if err != nil || f == nil {
		return basicfont.Face7x13
	}
	return f
}

func loadFace(path string) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	col, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, err
	}
	f, err := col.Font(0)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    facePoints,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

// Use returns face, or [Face] when face is nil.
func Use(face font.Face) font.Face {
	if face != nil {
		return face
	}
	return Face()
}

// Ascent is the face ascent in pixels.
func Ascent(face font.Face) int {
	return Use(face).Metrics().Ascent.Ceil()
}

// LineHeight is the recommended line spacing in pixels.
func LineHeight(face font.Face) int {
	m := Use(face).Metrics()
	h := (m.Ascent + m.Descent).Ceil()
	if extra := m.Height.Ceil(); extra > h {
		h = extra
	}
	if h < 1 {
		return 16
	}
	return h
}

// Advance is the pixel width of r.
func Advance(r rune, face font.Face) int {
	f := Use(face)
	a, ok := f.GlyphAdvance(r)
	if !ok {
		space, spaceOK := f.GlyphAdvance(' ')
		if spaceOK {
			a = space
		}
	}
	n := a.Ceil()
	if n < 1 {
		return 8
	}
	return n
}
