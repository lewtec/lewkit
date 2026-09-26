package icons

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// DecodeImage loads PNG or JPEG from path.
func DecodeImage(path string) (image.Image, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeBytes(raw, path)
}

// DecodeBytes decodes PNG/JPEG bytes. name is for error context only.
func DecodeBytes(raw []byte, name string) (image.Image, error) {
	r := bytes.NewReader(raw)
	img, format, err := image.Decode(r)
	if err != nil {
		// try jpeg explicitly if extension hints
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".jpg" || ext == ".jpeg" {
			if _, seekErr := r.Seek(0, 0); seekErr != nil {
				return nil, fmt.Errorf("decode %s: %w", name, seekErr)
			}
			return jpeg.Decode(r)
		}
		if ext == ".png" {
			if _, seekErr := r.Seek(0, 0); seekErr != nil {
				return nil, fmt.Errorf("decode %s: %w", name, seekErr)
			}
			return png.Decode(r)
		}
		return nil, fmt.Errorf("decode %s: %w (supported: png, jpeg)", name, err)
	}
	_ = format
	return img, nil
}

// PadCenter returns a square image: master centered, transparent margins if needed.
// Side is max(width, height).
func PadCenter(src image.Image) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h > side {
		side = h
	}
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	// transparent background (zero value)
	off := image.Pt((side-w)/2, (side-h)/2)
	r := image.Rect(off.X, off.Y, off.X+w, off.Y+h)
	draw.Draw(dst, r, src, b.Min, draw.Over)
	return dst
}

// KnockoutBackground makes pixels near the corner sample color transparent
// (soft edge). Used so launcher/splash marks are not sitting on a white box.
// Corners of photo-style logos are usually the canvas; solid brand marks are
// left alone if the corner is not near-white/near-uniform.
func KnockoutBackground(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	// Sample a few corners; use average if similar.
	samples := []image.Point{
		{b.Min.X, b.Min.Y},
		{b.Max.X - 1, b.Min.Y},
		{b.Min.X, b.Max.Y - 1},
		{b.Max.X - 1, b.Max.Y - 1},
	}
	var sr, sg, sb, n int
	for _, p := range samples {
		r, g, bl, a := src.At(p.X, p.Y).RGBA()
		if a < 0x8000 {
			continue // already transparent corner
		}
		sr += int(r >> 8)
		sg += int(g >> 8)
		sb += int(bl >> 8)
		n++
	}
	if n == 0 {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	cr, cg, cb := sr/n, sg/n, sb/n
	// Only knock out light canvases (avoid eating dark logos).
	if cr+cg+cb < 200*3 {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}

	const hard, soft = 28.0, 58.0
	hs, ss := hard*hard, soft*soft
	canvasDist := func(x, y int) (float64, int, int, int, int, bool) {
		r16, g16, b16, a16 := src.At(x, y).RGBA()
		r, g, bl, a := int(r16>>8), int(g16>>8), int(b16>>8), int(a16>>8)
		if a < 8 {
			return 0, r, g, bl, a, false
		}
		dr, dg, db := float64(r-cr), float64(g-cg), float64(bl-cb)
		d := dr*dr + dg*dg + db*db
		return d, r, g, bl, a, d < ss
	}

	// Only the canvas connected to the border. Interior highlights stay.
	w, h := b.Dx(), b.Dy()
	reach := make([]bool, w*h)
	idx := func(x, y int) int { return (x-b.Min.X) + (y-b.Min.Y)*w }
	q := make([]int, 0, w+h)
	push := func(x, y int) {
		if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
			return
		}
		i := idx(x, y)
		if reach[i] {
			return
		}
		if _, _, _, _, _, ok := canvasDist(x, y); !ok {
			return
		}
		reach[i] = true
		q = append(q, i)
	}
	for x := b.Min.X; x < b.Max.X; x++ {
		push(x, b.Min.Y)
		push(x, b.Max.Y-1)
	}
	for y := b.Min.Y + 1; y < b.Max.Y-1; y++ {
		push(b.Min.X, y)
		push(b.Max.X-1, y)
	}
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x := b.Min.X + i%w
		y := b.Min.Y + i/w
		for _, dxy := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			push(x+dxy[0], y+dxy[1])
		}
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d, r, g, bl, a, _ := canvasDist(x, y)
			ox, oy := x-b.Min.X, y-b.Min.Y
			if !reach[idx(x, y)] {
				r16, g16, b16, a16 := src.At(x, y).RGBA()
				dst.SetNRGBA(ox, oy, color.NRGBA{R: uint8(r16 >> 8), G: uint8(g16 >> 8), B: uint8(b16 >> 8), A: uint8(a16 >> 8)})
				continue
			}
			var alpha uint8
			switch {
			case d <= hs:
				alpha = 0
			default:
				t := (d - hs) / (ss - hs)
				alpha = uint8(float64(a) * t)
			}
			if alpha == 0 {
				dst.SetNRGBA(ox, oy, color.NRGBA{})
			} else {
				dst.SetNRGBA(ox, oy, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(bl), A: alpha})
			}
		}
	}
	return dst
}

// ContentBounds is the smallest rectangle covering pixels with alpha > minA
// (8-bit). If none qualify, it returns src.Bounds().
func ContentBounds(src image.Image, minA uint8) image.Rectangle {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if uint8(a>>8) <= minA {
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x+1 > maxX {
				maxX = x + 1
			}
			if y+1 > maxY {
				maxY = y + 1
			}
		}
	}
	if !found {
		return b
	}
	return image.Rect(minX, minY, maxX, maxY)
}

// cropTo copies r (intersected with src) into a 0,0-origin NRGBA.
func cropTo(src image.Image, r image.Rectangle) *image.NRGBA {
	r = r.Intersect(src.Bounds())
	dst := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	if r.Empty() {
		return dst
	}
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

// TrimTransparent crops to ContentBounds plus padFrac of the longer side,
// clipped to src. padFrac <= 0 means no extra padding.
func TrimTransparent(src image.Image, padFrac float64) *image.NRGBA {
	box := ContentBounds(src, 8)
	if padFrac > 0 {
		pad := int(float64(max(box.Dx(), box.Dy())) * padFrac)
		box = box.Inset(-pad)
	}
	return cropTo(src, box)
}

// largestEmptyRowRun returns the [start,end) empty-row span inside box with
// the most rows. ok is false when no empty row exists.
func largestEmptyRowRun(src image.Image, box image.Rectangle, minA uint8) (start, end int, ok bool) {
	best0, best1 := 0, 0
	run0 := -1
	for y := box.Min.Y; y < box.Max.Y; y++ {
		empty := true
		for x := box.Min.X; x < box.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if uint8(a>>8) > minA {
				empty = false
				break
			}
		}
		if empty {
			if run0 < 0 {
				run0 = y
			}
			continue
		}
		if run0 >= 0 && y-run0 > best1-best0 {
			best0, best1 = run0, y
		}
		run0 = -1
	}
	if run0 >= 0 && box.Max.Y-run0 > best1-best0 {
		best0, best1 = run0, box.Max.Y
	}
	if best1 <= best0 {
		return 0, 0, false
	}
	return best0, best1, true
}

// ExtractUpperMark crops the symbol above the largest transparent horizontal
// gap (lockup wordmark sits below). With no gap it uses the full content box.
// The result is a padded square.
func ExtractUpperMark(src image.Image) *image.NRGBA {
	const minA uint8 = 8
	box := ContentBounds(src, minA)
	mark := box
	if g0, g1, ok := largestEmptyRowRun(src, box, minA); ok {
		// Ignore hairline gaps; a lockup gap is a real band of empty rows.
		if g1-g0 >= max(8, box.Dy()/20) && g0 > box.Min.Y {
			mark.Max.Y = g0
		}
	}
	// Recompute x so leftover wordmark-width padding is not kept.
	mark = ContentBounds(cropTo(src, mark), minA).Add(mark.Min)
	const padFrac = 0.08
	pad := int(float64(max(mark.Dx(), mark.Dy())) * padFrac)
	return PadCenter(cropTo(src, mark.Inset(-pad)))
}

// Resize returns a size×size NRGBA using CatmullRom.
func Resize(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// EncodePNG encodes img as PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WritePNG writes a PNG file, creating parent dirs.
func WritePNG(path string, img image.Image) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return png.Encode(f, img)
}

// FlattenOpaque draws src onto white (for formats that dislike alpha).
func FlattenOpaque(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}
