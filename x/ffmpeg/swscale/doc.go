// Package swscale converts and scales pictures.
//
// The unscaled YUV to RGB path uses the FFmpeg 5.1.9 coefficient tables
// and the same lookup as libswscale's C yuv2rgb. Scaling is point,
// bilinear, bicubic, lanczos, or area.
package swscale
