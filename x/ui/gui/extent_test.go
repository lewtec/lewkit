package gui

import (
	"context"
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/present"
	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/stretchr/testify/require"
)

type androidHost struct{ window.Window }

func (androidHost) Surface() window.Surface {
	return window.Surface{Kind: window.SurfaceAndroid, A: 1}
}

// extentScreen reports the size the user sees. The swapchain image may be
// the identity size underneath that.
type extentScreen struct {
	oriented image.Point
	drawn    image.Point
	draws    int
	adopts   int
}

func (s *extentScreen) Oriented() image.Point { return s.oriented }

func (s *extentScreen) Draw(_ context.Context, _, _, _ []byte, width, height int) error {
	s.draws++
	s.drawn = image.Pt(width, height)
	return nil
}

func (s *extentScreen) Adopt(_ uintptr, _, _ int) error {
	s.adopts++
	return nil
}

func (*extentScreen) Close() error { return nil }

func TestLandscapeExtentRefitsBeforePresent(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 8, Height: 20, Period: time.Hour})
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })
	screen := &extentScreen{oriented: image.Pt(20, 8)}
	picture, err := NewPicture()
	require.NoError(t, err)
	paint := &sizedPaint{fill: RGB{255, 0, 0, 255}}
	runner := &runner{
		ctx:     t.Context(),
		host:    bridgeDisplay{Window: androidHost{host}, screen: screen},
		model:   paint,
		picture: picture,
		dirty:   true,
	}
	require.NoError(t, runner.flush(true))
	require.Equal(t, image.Pt(20, 8), host.Size())
	require.Equal(t, image.Pt(20, 8), paint.size())
	require.Equal(t, image.Pt(20, 8), screen.drawn)
	require.Equal(t, 1, screen.draws)
	require.GreaterOrEqual(t, screen.adopts, 2)
}

func TestDesktopExtentDoesNotResizeTheWindow(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 8, Height: 20, Period: time.Hour})
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })
	screen := &extentScreen{oriented: image.Pt(20, 8)}
	picture, err := NewPicture()
	require.NoError(t, err)
	runner := &runner{
		ctx:     t.Context(),
		host:    bridgeDisplay{Window: surfaceHost{host}, screen: screen},
		model:   &sizedPaint{fill: RGB{0, 255, 0, 255}},
		picture: picture,
		dirty:   true,
	}
	require.NoError(t, runner.flush(true))
	require.Equal(t, image.Pt(8, 20), host.Size())
	require.Equal(t, image.Pt(8, 20), screen.drawn)
	require.Equal(t, 1, screen.draws)
}

func TestMatchingAndroidExtentPaintsTheWindow(t *testing.T) {
	host, err := window.Open(t.Context(), window.Config{Width: 12, Height: 6, Period: time.Hour})
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })
	screen := &extentScreen{oriented: image.Pt(12, 6)}
	picture, err := NewPicture()
	require.NoError(t, err)
	runner := &runner{
		ctx:     t.Context(),
		host:    bridgeDisplay{Window: androidHost{host}, screen: screen},
		model:   &sizedPaint{fill: RGB{0, 0, 255, 255}},
		picture: picture,
		dirty:   true,
	}
	require.NoError(t, runner.flush(true))
	require.Equal(t, image.Pt(12, 6), host.Size())
	require.Equal(t, image.Pt(12, 6), screen.drawn)
	require.Equal(t, 1, screen.draws)
}

var _ present.Screen = (*extentScreen)(nil)
