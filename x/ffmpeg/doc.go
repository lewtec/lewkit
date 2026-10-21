// Package ffmpeg is the lewkit ffmpeg command.
//
// Run accepts the same arguments as the ffmpeg program. Version names the
// FFmpeg 9.0.2 tag placed at third_party/ffmpeg. The libraries under this
// tree are a Go transcription of libavutil, libavcodec, libavformat,
// libavfilter, libswscale, and libswresample, including the amd64 kernels
// those libraries use on the hot paths.
package ffmpeg
