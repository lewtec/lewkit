// Package htmx registers the vendored htmx script.
package htmx

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed htmx.min.js
var body []byte

// Version is the vendored htmx release.
const Version = "4.0.0"

// Name is the file under [asset.Prefix].
const Name = "htmx.min.js"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "text/javascript; charset=utf-8",
		Body:        body,
	})
}
