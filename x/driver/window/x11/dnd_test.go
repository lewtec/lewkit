package x11

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseURIList(t *testing.T) {
	raw := "file:///music/Hits/a.mp3\r\n# comment\r\nfile:///music/A%20B.wav\r\nhttp://example/x\r\n"
	assert.Equal(t, []string{"/music/Hits/a.mp3", "/music/A B.wav"}, parseURIList(raw))
}
