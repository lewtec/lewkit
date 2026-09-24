package image

import (
	"context"
	"os"
	"sync"

	"github.com/lewtec/lewkit/x/singleton"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
)

const facePoints = 18

var systemFont = singleton.NewSingleton(func(context.Context) (*opentype.Font, error) {
	var last error
	for _, path := range systemFonts {
		if path == "" {
			continue
		}
		f, err := openFont(path)
		if err == nil {
			return f, nil
		}
		last = err
	}
	if last == nil {
		last = os.ErrNotExist
	}
	return nil, last
})

var sizedFaces struct {
	sync.Mutex
	m map[int]font.Face
}

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

// Face is a system sans-serif face at 18px, or 7×13 if none is readable.
func Face() font.Face {
	return FaceSize(facePoints)
}

// FaceSize is [Face] at points pixels. The size is rounded to a whole number.
// A missing system font returns 7×13 at every size.
func FaceSize(points float64) font.Face {
	size := int(points + 0.5)
	if size < 8 {
		size = 8
	}
	if size > 256 {
		size = 256
	}
	parsed, err := singleton.Get(systemFont)
	if err != nil || parsed == nil {
		return basicfont.Face7x13
	}
	sizedFaces.Lock()
	defer sizedFaces.Unlock()
	if sizedFaces.m == nil {
		sizedFaces.m = map[int]font.Face{}
	}
	if face, ok := sizedFaces.m[size]; ok {
		return face
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return basicfont.Face7x13
	}
	sizedFaces.m[size] = face
	return face
}

func openFont(path string) (*opentype.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	col, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, err
	}
	return col.Font(0)
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
