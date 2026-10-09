//go:build !linux

package window

import "image"

// BundleIcon is the Linux bundle icon. Other hosts have none.
func BundleIcon() image.Image { return nil }

// ApplyXIcon sets an X11 window icon. Other hosts ignore it.
func ApplyXIcon(uint32, image.Image) error { return nil }
