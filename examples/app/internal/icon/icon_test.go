package icon

import (
	"testing"

	"github.com/lewtec/lewkit/x/http/asset/lucide"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func TestNodePaintsAndReuses(t *testing.T) {
	ink := gui.RGB{255, 0, 0, 255}
	node := Node(lucide.NameSearch, ink, 18)
	require.NotNil(t, node.Src)
	require.Equal(t, float32(18), node.Width)
	require.Equal(t, float32(18), node.Height)
	again := Node(lucide.NameSearch, ink, 12)
	require.Same(t, node.Src, again.Src)
}

func TestNodeUnknown(t *testing.T) {
	node := Node(lucide.Name("nope"), gui.RGB{0, 0, 0, 255}, 18)
	require.Nil(t, node.Src)
	again := Node(lucide.Name("nope"), gui.RGB{0, 0, 0, 255}, 18)
	require.Nil(t, again.Src)
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
		node := Node(name, ink, 18)
		require.NotNilf(t, node.Src, "%s", name)
	}
}
