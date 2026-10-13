//go:build !linux || android

package window

import "image"

// BundleIcon is the Linux bundle icon. Other hosts have none.
func BundleIcon() image.Image { return nil }
