// Package image packs ndarray pictures into image.RGBA.
//
// Pixels are shape (h, w, 4) RGBA in 0..255. [Fill] builds a float32
// tensor. [Eval] casts to uint8 and runs into an image.RGBA; [Write]
// copies a uint8 buffer.
package image
