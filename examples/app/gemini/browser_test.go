package main

import (
	"context"
	"image"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/net/gemini"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserBootAndRouteTransition(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)

	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	runCmd(t, browser, ctx, cmd)
	assert.Equal(t, "gemini://home/", browser.current.url)
	assert.Equal(t, "Geminispace Observatory", browser.current.title)
	assert.NotEmpty(t, browser.links)

	// Navigate to the spec page and verify the route transition tween starts at 0 and advances.
	navCmd := browser.navigate("gemini://home/spec", 1, true)
	runCmd(t, browser, ctx, navCmd)
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
	runCmd(t, browser, ctx, backCmd)
	assert.Equal(t, "gemini://home/", browser.current.url)
	assert.Equal(t, -1, browser.routeDir)
}

func TestBrowserStatus10SearchPromptAndDrawer(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)

	// Boot home first.
	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	runCmd(t, browser, ctx, cmd)

	// Status 10 asks the prompt driver, then opens the query URL.
	var prompted string
	browser.askFn = func(_ context.Context, prompt string) (string, error) {
		prompted = prompt
		return "spring", nil
	}
	searchCmd := browser.navigate("gemini://home/search", 1, true)
	settle(t, browser, ctx, searchCmd)
	assert.Contains(t, prompted, "search")
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

func TestNavigateAnimatesWhileFetching(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)
	browser.booted = true
	release := make(chan struct{})
	browser.Client = &gemini.Client{
		Hosts: map[string]gemini.Handler{
			"slow": func(ctx context.Context, _ *url.URL) (gemini.Response, error) {
				select {
				case <-release:
				case <-ctx.Done():
					return gemini.Response{}, ctx.Err()
				}
				return gemini.Response{Status: gemini.CodeSuccess, Meta: "text/gemini", Body: []byte("# hi\n")}, nil
			},
		},
	}
	cmd := browser.navigate("gemini://slow/", 1, true)
	require.NotNil(t, cmd)
	_, cmd = browser.Update(cmd(ctx))
	assert.True(t, browser.loading)
	before := browser.loadPhase
	for i := 0; i < 3; i++ {
		require.NotNil(t, cmd)
		_, cmd = browser.Update(cmd(ctx))
	}
	assert.NotEqual(t, before, browser.loadPhase)
	assert.True(t, browser.loading)
	close(release)
	runCmd(t, browser, ctx, gui.Tick())
	assert.False(t, browser.loading)
	assert.Equal(t, "gemini://slow/", browser.current.url)
}

func TestSwipeScrollsWithoutLeavingThePage(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)
	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	runCmd(t, browser, ctx, cmd)
	url := browser.current.url

	_, _ = browser.Update(window.Pointer{Pos: image.Pt(200, 500), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	_, cmd = browser.Update(window.Pointer{Pos: image.Pt(200, 320), Buttons: window.ButtonLeft})
	require.NotNil(t, cmd)
	assert.Greater(t, browser.scrollSpring.Target, float32(100))
	_, cmd = browser.Update(window.Pointer{Pos: image.Pt(200, 320), Button: 1})
	assert.Nil(t, cmd)
	assert.Equal(t, url, browser.current.url)

	root := browser.View()
	size := gui.Size{Width: 960, Height: 700}
	var at image.Point
	for y := 0; y < 700 && at == (image.Point{}); y += 6 {
		key, _, _ := gui.Hit(root, size, image.Pt(180, y))
		if strings.HasPrefix(key, "link:") {
			at = image.Pt(180, y)
		}
	}
	require.NotEqual(t, image.Point{}, at)
	_, _ = browser.Update(window.Pointer{Pos: at, Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	_, cmd = browser.Update(window.Pointer{Pos: at, Button: 1})
	runCmd(t, browser, ctx, cmd)
	assert.NotEqual(t, url, browser.current.url)
}

func TestAddressUsesPrompt(t *testing.T) {
	ctx := t.Context()
	browser, err := NewBrowser(ctx)
	require.NoError(t, err)
	_, cmd := browser.Update(gui.TickMsg{Elapsed: 16 * time.Millisecond, Size: image.Pt(960, 700)})
	runCmd(t, browser, ctx, cmd)

	browser.askFn = func(_ context.Context, prompt string) (string, error) {
		require.Equal(t, "Address", prompt)
		return "gemini://home/spec", nil
	}
	_, cmd = browser.Update(window.Key{Pressed: true, Rune: '/'})
	settle(t, browser, ctx, cmd)
	assert.Equal(t, "gemini://home/spec", browser.current.url)

	browser.Client.Hosts["lucasew.srht.site"] = func(context.Context, *url.URL) (gemini.Response, error) {
		return gemini.Response{Status: gemini.CodeSuccess, Meta: "text/gemini", Body: []byte("# Agena\n")}, nil
	}
	browser.askFn = func(context.Context, string) (string, error) {
		return "lucasew.srht.site", nil
	}
	_, cmd = browser.Update(window.Key{Pressed: true, Rune: '/'})
	settle(t, browser, ctx, cmd)
	assert.Equal(t, "gemini://lucasew.srht.site/", browser.current.url)
	assert.Equal(t, "Agena", browser.current.title)

	browser.askFn = func(context.Context, string) (string, error) {
		return "", errs.ErrCanceled
	}
	_, cmd = browser.Update(window.Key{Pressed: true, Rune: '/'})
	require.NotNil(t, cmd)
	_, _ = browser.Update(cmd(ctx))
	assert.Equal(t, "gemini://lucasew.srht.site/", browser.current.url)
}

func settle(t *testing.T, browser *Browser, ctx context.Context, cmd gui.Cmd) {
	t.Helper()
	var msg gui.Msg
	if cmd != nil {
		msg = cmd(ctx)
	}
	require.Eventually(t, func() bool {
		if msg != nil {
			_, cmd = browser.Update(msg)
			msg = nil
		} else if browser.loading {
			_, cmd = browser.Update(gui.TickMsg{Elapsed: 20 * time.Millisecond, Size: image.Pt(960, 700)})
		}
		if cmd != nil {
			msg = cmd(ctx)
			cmd = nil
			return false
		}
		return !browser.loading
	}, 2*time.Second, time.Millisecond)
}

func runCmd(t *testing.T, browser *Browser, ctx context.Context, cmd gui.Cmd) {
	t.Helper()
	require.NotNil(t, cmd)
	msg := cmd(ctx)
	require.Eventually(t, func() bool {
		if msg != nil {
			_, cmd = browser.Update(msg)
			msg = nil
		} else if browser.loading {
			_, cmd = browser.Update(gui.TickMsg{Elapsed: 20 * time.Millisecond, Size: image.Pt(960, 700)})
		}
		if cmd != nil {
			msg = cmd(ctx)
			cmd = nil
		}
		return !browser.loading && browser.current.url != ""
	}, 2*time.Second, time.Millisecond)
}
