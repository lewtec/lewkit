// Package image draws into an ndarray: one fused kernel per picture.
//
// Pixels are shape (h, w, 4) float32 RGBA in 0..255. [Triangle] and [Fill]
// return a [github.com/lewtec/lewkit/x/ndarray.Tensor]. [Painter] keeps one
// device and reuses the compiled kernel across frames. [RGBA] packs the
// dense output into an image.RGBA.
package image
