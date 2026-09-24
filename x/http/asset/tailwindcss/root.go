// Package tailwindcss registers the vendored Tailwind browser script.
package tailwindcss

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed tailwindcss.js
var body []byte

// Version is the vendored @tailwindcss/browser release.
const Version = "4.3.3"

// Name is the file under [asset.Prefix].
const Name = "tailwindcss.js"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "text/javascript; charset=utf-8",
		Body:        body,
	})
}
