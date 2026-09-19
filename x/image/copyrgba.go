package image

import "image"

// CopyRGBA copies packed RGBA8 rows (dst width × height × 4) into dst.
func CopyRGBA(dst *image.RGBA, src []byte) {
	if dst == nil {
		return
	}
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	need := w * h * 4
	if w < 1 || h < 1 || len(src) < need {
		return
	}
	for y := range h {
		row := src[y*w*4 : (y+1)*w*4]
		copy(dst.Pix[y*dst.Stride:], row)
	}
}
