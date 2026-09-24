package x11

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeysymRune(t *testing.T) {
	assert.Equal(t, 'a', keysymRune(0x61))
	assert.Equal(t, 'M', keysymRune(0x4d))
	assert.Equal(t, rune(8), keysymRune(0xff08))
	assert.Equal(t, '\n', keysymRune(0xff0d))
	assert.Equal(t, rune(0), keysymRune(0xff51))
}
