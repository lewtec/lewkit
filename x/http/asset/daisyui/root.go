// Package daisyui registers the vendored daisyUI stylesheet.
package daisyui

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed daisyui.css
var body []byte

// Version is the vendored daisyUI release.
const Version = "5.6.18"

// Name is the file under [asset.Prefix].
const Name = "daisyui.css"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "text/css; charset=utf-8",
		Body:        body,
	})
}
