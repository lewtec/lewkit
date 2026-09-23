// Package jquery registers the vendored jQuery script.
package jquery

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed jquery.min.js
var body []byte

// Version is the vendored jQuery release.
const Version = "4.0.0"

// Name is the file under [asset.Prefix].
const Name = "jquery.min.js"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "text/javascript; charset=utf-8",
		Body:        body,
	})
}
