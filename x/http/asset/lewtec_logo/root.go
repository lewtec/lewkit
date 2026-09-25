// Package lewtec_logo registers the LEWTEC TECNOLOGIA lockup.
package lewtec_logo

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed logo_full.png
var body []byte

// Name is the file under [asset.Prefix].
const Name = "logo_full.png"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "image/png",
		Body:        body,
	})
}

// Image is the lockup. It is the raster of monorepo/branding/logo_full.svg.
var picture = mustImage()

func mustImage() image.Image {
	decoded, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	return decoded
}

func Image() image.Image { return picture }
