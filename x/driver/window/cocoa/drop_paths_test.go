package cocoa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDropPaths(t *testing.T) {
	got := dropPaths([]string{
		"file:///Users/ada/Music/Hits",
		"file:///Users/ada/A%20B.wav",
		"/tmp/song.mp3",
		"http://example/x",
		"",
	})
	assert.Equal(t, []string{"/Users/ada/Music/Hits", "/Users/ada/A B.wav", "/tmp/song.mp3"}, got)
}
