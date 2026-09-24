package gui

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

//go:embed logo_full.png
var logoPNG []byte

// logoImage is the LEWTEC TECNOLOGIA lockup rasterized from monorepo/branding/logo_full.svg.
var logoDecoded = mustLogo()

func mustLogo() image.Image {
	decoded, err := png.Decode(bytes.NewReader(logoPNG))
	if err != nil {
		panic(err)
	}
	return decoded
}

func logoImage() image.Image { return logoDecoded }
