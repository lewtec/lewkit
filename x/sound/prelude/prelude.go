// Package prelude registers the audio decoders via blank imports.
package prelude

import (
	_ "github.com/lewtec/lewkit/x/sound/mp3"
	_ "github.com/lewtec/lewkit/x/sound/ogg"
)
