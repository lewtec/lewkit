package gui

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
)

func TestEnterKey(t *testing.T) {
	assert.True(t, enterKey(window.Key{Rune: '\r'}))
	assert.True(t, enterKey(window.Key{Rune: '\n'}))
	assert.True(t, enterKey(window.Key{Code: 36}))
	assert.True(t, enterKey(window.Key{Code: 13}))
	assert.True(t, enterKey(window.Key{Code: 24}))
	assert.False(t, enterKey(window.Key{Rune: 'a', Code: 0}))
}
