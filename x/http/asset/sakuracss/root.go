// Package sakuracss registers the vendored Sakura CSS stylesheet.
package sakuracss

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed sakura.css
var body []byte

// Version is the vendored Sakura CSS release.
const Version = "1.5.1"

// Name is the file under [asset.Prefix].
const Name = "sakura.css"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "text/css; charset=utf-8",
		Body:        body,
	})
}
