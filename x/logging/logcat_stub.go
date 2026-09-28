//go:build !android

package logging

import "io"

// Logcat is the Android log. It is absent on this platform.
func Logcat() io.Writer { return nil }
