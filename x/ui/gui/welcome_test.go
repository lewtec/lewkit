package gui

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWelcomePicksRecent(t *testing.T) {
	dir := t.TempDir()
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit", Dirs: []Directory{{Path: dir}, {Path: t.TempDir()}}})
	paintWelcome(t, welcome)
	require.Len(t, welcome.rows, 2)
	_, cmd := welcome.Update(window.Pointer{Pos: mid(welcome.rows[0]), Button: 1, Pressed: true})
	assertQuit(t, cmd)
	assert.Equal(t, dir, welcome.Picked())
}

func TestWelcomeKeyboard(t *testing.T) {
	second := t.TempDir()
	welcome := NewWelcome(WelcomeArgs{Dirs: []Directory{{Path: t.TempDir()}, {Path: second}}})
	welcome.Update(window.Key{Code: 116, Pressed: true})
	welcome.Update(window.Key{Rune: '\n', Pressed: true})
	assert.Equal(t, second, welcome.Picked())
	assert.Equal(t, "lewkit", welcome.title)
}

func TestWelcomeEscape(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	_, cmd := welcome.Update(window.Key{Rune: 0x1b, Pressed: true})
	assertQuit(t, cmd)
	assert.Empty(t, welcome.Picked())
}

func TestWelcomeDialogUsesCallerContext(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	welcome.cursor = welcome.browseAt()
	_, cmd := welcome.Update(window.Key{Rune: '\n', Pressed: true})
	require.NotNil(t, cmd)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	picked, ok := cmd(ctx).(folderPicked)
	require.True(t, ok)
	assert.ErrorIs(t, picked.err, context.Canceled)
}

func TestWelcomeOpenCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	done := make(chan error, 1)
	go func() {
		done <- Open(ctx, welcome, Options{Config: window.Config{Width: 80, Height: 60}})
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
		assert.Empty(t, welcome.Picked())
	case <-time.After(2 * time.Second):
		t.Fatal("Open did not return")
	}
}

func TestWelcomeLightMode(t *testing.T) {
	accent := Color{200, 0, 0, 255}
	welcome := NewWelcome(WelcomeArgs{
		Title:  "lewkit",
		Dirs:   []Directory{{Path: t.TempDir()}},
		Accent: &accent,
	})
	welcome.Update(ModeMsg{Mode: daynight.Light})
	node := welcome.View()
	require.NotNil(t, node)
	card, selected := welcome.cards()
	assert.Equal(t, Color{255, 255, 255, 255}, card)
	assert.Greater(t, int(selected.Red), int(selected.Blue))
}

func TestWelcomeAccentFromLogo(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit", Dirs: []Directory{{Path: t.TempDir()}}})
	welcome.View()
	require.NotNil(t, welcome.browse.Fill)
	assert.Equal(t, lewimage.Average(logoImage()), *welcome.browse.Fill)
}

func TestWelcomeAccentOverride(t *testing.T) {
	accent := Color{20, 180, 40, 255}
	welcome := NewWelcome(WelcomeArgs{
		Title:  "lewkit",
		Dirs:   []Directory{{Path: t.TempDir()}},
		Accent: &accent,
	})
	welcome.View()
	assert.Equal(t, accent, *welcome.browse.Fill)
}

func TestWelcomeLogo(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	_, err = picture.Render(welcome.View(), Size{Width: 880, Height: 720})
	require.NoError(t, err)
	require.NotEmpty(t, picture.images)
	var navy int
	for i := 0; i+3 < len(picture.inkRGBA.Pix); i += 4 {
		red, green, blue, alpha := picture.inkRGBA.Pix[i], picture.inkRGBA.Pix[i+1], picture.inkRGBA.Pix[i+2], picture.inkRGBA.Pix[i+3]
		if alpha > 200 && blue > red && blue > green && blue > 40 {
			navy++
		}
	}
	assert.Greater(t, navy, 50)
}

func TestImageKeepsStraightAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	src.Pix = []uint8{13, 53, 89, 128}
	node := &Image{Src: src, Width: 1, Height: 1}
	node.Layout(Tight(1, 1))
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	node.Paint(Offset{}, Rect{0, 0, 1, 1}, picture)
	require.NoError(t, picture.paintInk(1, 1))
	assert.Equal(t, []uint8{13, 53, 89, 128}, picture.inkRGBA.Pix[:4])
}

func TestImageClip(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	node := &Image{Src: src, Width: 8, Height: 8}
	node.Layout(Tight(8, 8))
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	node.Paint(Offset{}, Rect{0, 0, 3, 8}, picture)
	require.NoError(t, picture.paintInk(8, 8))
	pix := picture.inkRGBA.Pix
	assert.Equal(t, uint8(255), pix[0])
	assert.Equal(t, uint8(0), pix[3*4])
}

func assertQuit(t *testing.T, cmd Cmd) {
	t.Helper()
	require.NotNil(t, cmd)
	_, ok := cmd(context.Background()).(quitMsg)
	assert.True(t, ok)
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
