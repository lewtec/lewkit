//go:build !android

package app

import (
	"image"
	"sync"

	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/driver/window"
)

var (
	markOnce sync.Once
	markImg  image.Image
)

func shellMark() image.Image {
	markOnce.Do(func() {
		if img := window.BundleIcon(); img != nil {
			markImg = img
			return
		}
		img, err := icons.DefaultMark()
		if err != nil {
			return
		}
		markImg = img
	})
	return markImg
}
