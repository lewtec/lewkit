// Package hastad_nha registers the vendored Hastad Nha clip.
package hastad_nha

import (
	_ "embed"

	"github.com/lewtec/lewkit/x/http/asset"
)

//go:embed hastad-nha.mp3
var body []byte

// Name is the file under [asset.Prefix].
const Name = "hastad-nha.mp3"

// Path is the URL a page loads.
const Path = asset.Prefix + Name

func init() {
	asset.MustRegister(asset.File{
		Name:        Name,
		ContentType: "audio/mpeg",
		Body:        body,
	})
}

// Bytes is the MP3. Decode it with sound.Decode. Do not modify the slice.
func Bytes() []byte { return body }
