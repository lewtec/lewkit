package main

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserBootAndRouteTransition(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)

	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	require.NotNil(t, cmd)
	msg := cmd(ctx)
	_, cmd = browser.Update(msg)
	require.NotNil(t, cmd)
	assert.Equal(t, "gemini://home/", browser.current.url)
	assert.Equal(t, "Geminispace Observatory", browser.current.title)
	assert.NotEmpty(t, browser.links)

	// Navigate to the spec page and verify the route transition tween starts at 0 and advances.
	navCmd := browser.navigate("gemini://home/spec", 1, true)
	require.NotNil(t, navCmd)
	_, _ = browser.Update(navCmd(ctx))
	assert.Equal(t, "gemini://home/spec", browser.current.url)
	require.NotNil(t, browser.previous)
	assert.Equal(t, "gemini://home/", browser.previous.url)
	assert.True(t, browser.routeProgress.Animating())

	// Step a few ticks and render each frame through Picture.
	picture, err := gui.NewPicture()
	require.NoError(t, err)
	picture.SetScale(1)
	for i := 1; i <= 25; i++ {
		_, _ = browser.Update(gui.TickMsg{
			Elapsed: time.Duration(i+1) * 16 * time.Millisecond,
			Size:    image.Pt(960, 700),
		})
		_, renderErr := picture.Render(browser.View(), gui.Size{Width: 960, Height: 700})
		require.NoError(t, renderErr)
	}
	assert.False(t, browser.routeProgress.Animating())
	assert.Nil(t, browser.previous)

	// Go back in history and verify reverse transition.
	_, backCmd := browser.goBack()
	require.NotNil(t, backCmd)
	_, _ = browser.Update(backCmd(ctx))
	assert.Equal(t, "gemini://home/", browser.current.url)
	assert.Equal(t, -1, browser.routeDir)
}

func TestBrowserStatus10SearchPromptAndDrawer(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)

	// Boot home first.
	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	_, _ = browser.Update(cmd(ctx))

	// Navigate to gemini://home/search which returns Status 10 INPUT.
	searchCmd := browser.navigate("gemini://home/search", 1, true)
	_, _ = browser.Update(searchCmd(ctx))
	assert.True(t, browser.promptOpen)
	assert.True(t, browser.promptScale.Animating())

	// Type "spring" and press Enter.
	for _, r := range "spring" {
		_, _ = browser.Update(window.Key{Pressed: true, Rune: r})
	}
	_, submitCmd := browser.Update(window.Key{Pressed: true, Rune: '\n'})
	require.NotNil(t, submitCmd)
	_, _ = browser.Update(submitCmd(ctx))
	assert.False(t, browser.promptOpen)
	assert.Equal(t, "gemini://home/search?spring", browser.current.url)
	assert.Contains(t, browser.current.title, "Search Results")

	// Open drawer, toggle theme, and scroll with spring physics.
	_, _ = browser.Update(window.Key{Pressed: true, Rune: 'b'})
	assert.True(t, browser.drawerOpen)
	assert.True(t, browser.drawerSlide.Animating())

	_, _ = browser.Update(gui.ModeMsg{Mode: daynight.Light})
	assert.True(t, browser.themeMix.Animating())

	_, _ = browser.Update(window.Scroll{Delta: image.Pt(0, -3)})
	assert.Greater(t, browser.scrollSpring.Target, float32(0))
}
