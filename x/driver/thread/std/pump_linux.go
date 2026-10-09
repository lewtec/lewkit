//go:build linux

package std

import "github.com/lewtec/lewkit/x/ffi/native/gtk"

// pumpMessages drains the GTK queue when this thread owns it.
// Dialogs created here dispatch while Loop is between jobs.
func pumpMessages() { gtk.Poll() }
