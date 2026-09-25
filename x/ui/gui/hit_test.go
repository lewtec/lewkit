package gui

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHitFindsTopKey(t *testing.T) {
	root := &Box{
		Width: 40, Height: 30, Key: "outer", Fill: &RGB{1, 1, 1, 255},
		Child: &Box{Width: 10, Height: 10, Key: "inner", Fill: &RGB{2, 0, 0, 255}},
	}
	key, along, _ := Hit(root, Size{40, 30}, image.Pt(2, 2))
	assert.Equal(t, "inner", key)
	assert.InDelta(t, 0.2, along, 0.05)
	key, _, _ = Hit(root, Size{40, 30}, image.Pt(30, 20))
	assert.Equal(t, "outer", key)
	require.NotEmpty(t, key)
}
