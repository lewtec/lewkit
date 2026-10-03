package window_test

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
)

func TestTouchPointerDragKeepsLeftButton(t *testing.T) {
	down := window.TouchPointer(3, 4, 0)
	move := window.TouchPointer(3, 40, 2)
	up := window.TouchPointer(3, 40, 1)
	assert.Equal(t, window.Pointer{Pos: image.Pt(3, 4), Button: 1, Pressed: true, Buttons: window.ButtonLeft}, down)
	assert.Equal(t, window.Pointer{Pos: image.Pt(3, 40), Buttons: window.ButtonLeft}, move)
	assert.Equal(t, window.Pointer{Pos: image.Pt(3, 40), Button: 1}, up)
}
