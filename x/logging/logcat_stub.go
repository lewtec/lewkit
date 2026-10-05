//go:build !android || !cgo

package logging

import "io"

// Logcat is the Android log. It is nil unless this is an android cgo build:
// __android_log_write is only reachable through cgo.
func Logcat() io.Writer { return nil }
