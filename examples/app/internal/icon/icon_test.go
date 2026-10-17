package icon

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/http/asset/lucide"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func TestNodePaintsAtItsSizeAndReuses(t *testing.T) {
	ink := gui.RGB{255, 0, 0, 255}
	node := laidOut(t, lucide.NameSearch, ink, 18)
	require.Equal(t, image.Rect(0, 0, 18, 18), node.image.Src.Bounds())
	require.Equal(t, float32(18), node.image.Width)
	again := laidOut(t, lucide.NameSearch, ink, 18)
	require.Same(t, node.image.Src, again.image.Src)
	small := laidOut(t, lucide.NameSearch, ink, 12)
	require.Equal(t, image.Rect(0, 0, 12, 12), small.image.Src.Bounds())
	require.NotSame(t, node.image.Src, small.image.Src)
}

func TestIconPixelsFollowsDensity(t *testing.T) {
	require.Equal(t, 18, iconPixels(18, 1))
	require.Equal(t, 36, iconPixels(18, 2))
	require.Equal(t, 1, iconPixels(0, 1))
}

func TestNodeUnknown(t *testing.T) {
	node := laidOut(t, lucide.Name("nope"), gui.RGB{0, 0, 0, 255}, 18)
	require.Nil(t, node.image.Src)
	again := laidOut(t, lucide.Name("nope"), gui.RGB{0, 0, 0, 255}, 18)
	require.Nil(t, again.image.Src)
}

func TestChromeIconsPaint(t *testing.T) {
	ink := gui.RGB{238, 238, 238, 255}
	names := []lucide.Name{
		lucide.NameChevronLeft,
		lucide.NameChevronRight,
		lucide.NameHouse,
		lucide.NameRotateCcw,
		lucide.NameBookmark,
		lucide.NameBookmarkCheck,
		lucide.NameSun,
		lucide.NameMoon,
		lucide.NameMenu,
		lucide.NameSearch,
		lucide.NamePlay,
		lucide.NamePause,
		lucide.NameSkipBack,
		lucide.NameSkipForward,
		lucide.NamePlus,
		lucide.NameMinus,
		lucide.NameArrowRight,
		lucide.NameCircleSmall,
	}
	for _, name := range names {
		node := laidOut(t, name, ink, 18)
		require.NotNilf(t, node.image.Src, "%s", name)
		require.Equal(t, image.Rect(0, 0, 18, 18), node.image.Src.Bounds())
	}
}

func laidOut(t *testing.T, name lucide.Name, ink gui.RGB, size float32) *glyph {
	t.Helper()
	node := Node(name, ink, size).(*glyph)
	node.Layout(gui.Tight(400, 400))
	return node
}
