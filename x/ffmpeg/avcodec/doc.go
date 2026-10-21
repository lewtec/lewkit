// Package avcodec decodes and encodes the codecs this tree implements.
// PCM conversion follows libavcodec/pcm.c. Image codecs use the Go standard
// library for PNG, JPEG, and GIF. FLAC reads FFmpeg 5.1.9 frames, including
// fixed and LPC subframes, and writes verbatim FLAC.
package avcodec
