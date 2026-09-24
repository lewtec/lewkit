package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/http/asset/logo"
)

func logoImage() image.Image { return logo.Image() }
