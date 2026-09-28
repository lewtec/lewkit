//go:build !android

package android

import "errors"

// ErrUnavailable means this build has no Android loader.
var ErrUnavailable = errors.New("android binding unavailable")

// JavaVMs is unavailable outside Android.
func JavaVMs() (int, error) { return 0, ErrUnavailable }

// OnLooper is unavailable outside Android.
func OnLooper() (bool, error) { return false, ErrUnavailable }
