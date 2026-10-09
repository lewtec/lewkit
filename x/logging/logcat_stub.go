//go:build !android

package logging

import "io"

// Logcat is the Android log. It is nil on hosts that are not Android.
func Logcat() io.Writer { return nil }
