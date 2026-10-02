//go:build !darwin && !windows

package window

import "image"

func showShell(string, image.Image) {}
