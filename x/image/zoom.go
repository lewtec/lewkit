package image

import (
	stdimage "image"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Zoom returns face drawn at scale. One, or a missing scale, returns [Use] of face.
// A nil face at any other scale is [FaceSize] of 18 times that scale.
// Any other face is a bitmap enlargement of its glyphs.
func Zoom(face font.Face, scale float64) font.Face {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) || math.Abs(scale-1) <= 1e-3 {
		return Use(face)
	}
	if face == nil {
		return FaceSize(facePoints * scale)
	}
	return &zoomFace{base: face, scale: scale}
}

type zoomFace struct {
	base  font.Face
	scale float64
}

func (z *zoomFace) Close() error { return nil }

func (z *zoomFace) Glyph(dot fixed.Point26_6, r rune) (stdimage.Rectangle, stdimage.Image, stdimage.Point, fixed.Int26_6, bool) {
	bounds, mask, maskp, advance, ok := z.base.Glyph(fixed.P(0, 0), r)
	advance = scaleFixed(advance, z.scale)
	if mask == nil || bounds.Empty() {
		return stdimage.Rectangle{}, nil, stdimage.Point{}, advance, ok
	}
	src := stdimage.Rect(maskp.X, maskp.Y, maskp.X+bounds.Dx(), maskp.Y+bounds.Dy()).Intersect(mask.Bounds())
	if src.Empty() {
		return stdimage.Rectangle{}, nil, stdimage.Point{}, advance, ok
	}
	width := zoomDim(bounds.Dx(), z.scale)
	height := zoomDim(bounds.Dy(), z.scale)
	dst := stdimage.NewAlpha(stdimage.Rect(0, 0, width, height))
	z.scaler().Scale(dst, dst.Bounds(), mask, src, draw.Src, nil)
	ox := zoomOffset(dot.X, bounds.Min.X, z.scale)
	oy := zoomOffset(dot.Y, bounds.Min.Y, z.scale)
	return stdimage.Rect(ox, oy, ox+width, oy+height), dst, stdimage.Point{}, advance, ok
}

func (z *zoomFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	bounds, advance, ok := z.base.GlyphBounds(r)
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: scaleFixed(bounds.Min.X, z.scale), Y: scaleFixed(bounds.Min.Y, z.scale)},
		Max: fixed.Point26_6{X: scaleFixed(bounds.Max.X, z.scale), Y: scaleFixed(bounds.Max.Y, z.scale)},
	}, scaleFixed(advance, z.scale), ok
}

func (z *zoomFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	advance, ok := z.base.GlyphAdvance(r)
	return scaleFixed(advance, z.scale), ok
}

func (z *zoomFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return scaleFixed(z.base.Kern(r0, r1), z.scale)
}

func (z *zoomFace) Metrics() font.Metrics {
	metrics := z.base.Metrics()
	return font.Metrics{
		Height:     scaleFixed(metrics.Height, z.scale),
		Ascent:     scaleFixed(metrics.Ascent, z.scale),
		Descent:    scaleFixed(metrics.Descent, z.scale),
		XHeight:    scaleFixed(metrics.XHeight, z.scale),
		CapHeight:  scaleFixed(metrics.CapHeight, z.scale),
		CaretSlope: metrics.CaretSlope,
	}
}

func (z *zoomFace) scaler() xdraw.Scaler {
	if math.Abs(z.scale-math.Round(z.scale)) <= 0.01 {
		return xdraw.NearestNeighbor
	}
	return xdraw.ApproxBiLinear
}

func scaleFixed(v fixed.Int26_6, scale float64) fixed.Int26_6 {
	return fixed.Int26_6(math.Round(float64(v) * scale))
}

func zoomDim(n int, scale float64) int {
	v := int(math.Round(float64(n) * scale))
	if v < 1 {
		return 1
	}
	return v
}

func zoomOffset(dot fixed.Int26_6, pixels int, scale float64) int {
	return int(math.Round(float64(dot)/64 + float64(pixels)*scale))
}
