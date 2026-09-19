// Package image packs ndarray pictures into image.RGBA.
//
// Pixels are shape (h, w, 4) float32 RGBA in 0..255. [Fill] builds a
// constant tensor. [Eval] runs a tensor into an image.RGBA; [Write]
// packs a float buffer.
package image
