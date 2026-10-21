//go:build android

package app

import (
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
)

func shellMark() image.Image { return window.BundleIcon() }
