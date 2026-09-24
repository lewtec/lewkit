package gui

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWelcomePicksRecent(t *testing.T) {
	dir := t.TempDir()
	welcome := NewWelcome("lewkit", []Directory{{Path: dir}, {Path: t.TempDir()}})
	paintWelcome(t, welcome)
	require.Len(t, welcome.rows, 2)
	_, cmd := welcome.Update(window.Pointer{Pos: mid(welcome.rows[0]), Button: 1, Pressed: true})
	assert.Nil(t, cmd)
	assert.Equal(t, dir, welcome.Picked())
}

func TestWelcomeKeyboard(t *testing.T) {
	second := t.TempDir()
	welcome := NewWelcome("", []Directory{{Path: t.TempDir()}, {Path: second}})
	welcome.Update(window.Key{Code: 116, Pressed: true})
	welcome.Update(window.Key{Rune: '\n', Pressed: true})
	assert.Equal(t, second, welcome.Picked())
	assert.Equal(t, "lewkit", welcome.title)
}

func TestWelcomeEscape(t *testing.T) {
	var stopped bool
	welcome := NewWelcome("lewkit", nil)
	welcome.OnDone(func() { stopped = true })
	welcome.Update(window.Key{Rune: 0x1b, Pressed: true})
	assert.True(t, stopped)
	assert.Empty(t, welcome.Picked())
}

func TestWelcomeLightMode(t *testing.T) {
	welcome := NewWelcome("lewkit", []Directory{{Path: t.TempDir()}})
	welcome.Update(ModeMsg{Mode: daynight.Light})
	node := welcome.View()
	require.NotNil(t, node)
	card, selected := welcome.cards()
	assert.Equal(t, Color{255, 255, 255, 255}, card)
	assert.Equal(t, Color{255, 228, 186, 255}, selected)
}

func TestWelcomeMarkPaints(t *testing.T) {
	welcome := NewWelcome("lewkit", nil)
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	_, err = picture.Render(welcome.View(), Size{Width: 880, Height: 720})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(picture.fills), 3)
}

func paintWelcome(t *testing.T, welcome *Welcome) {
	t.Helper()
	node := welcome.View()
	node.Layout(Tight(880, 720))
	node.Paint(Offset{}, Rect{0, 0, 880, 720}, nil)
}

func mid(box *Box) image.Point {
	return image.Pt(int(box.origin.X+box.size.Width/2), int(box.origin.Y+box.size.Height/2))
}
