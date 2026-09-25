package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/http/asset/lewtec_logo"
)

func logoImage() image.Image { return lewtec_logo.Image() }
