// Package swresample converts PCM sample format, channel layout, and rate.
// Integer and float conversions follow libswresample/audioconvert.c from
// FFmpeg 5.1.9. Rate change is linear interpolation in float.
package swresample
