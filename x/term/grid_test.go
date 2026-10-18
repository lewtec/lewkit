package term

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridFeedAndReflow(t *testing.T) {
	grid := NewGrid(4, 4)
	grid.Feed([]byte("abcdefgh"))
	assert.Equal(t, "abcd\nefgh", grid.Text())

	grid.Reflow(4, 8)
	assert.Equal(t, "abcdefgh", grid.Text())

	grid.Reflow(4, 3)
	assert.Equal(t, "abc\ndef\ngh", grid.Text())
}

func TestGridNewlineDoesNotJoin(t *testing.T) {
	grid := NewGrid(4, 8)
	grid.Feed([]byte("ab\ncd"))
	assert.Equal(t, "ab\ncd", grid.Text())
	grid.Reflow(4, 8)
	assert.Equal(t, "ab\ncd", grid.Text())
}

func TestGridReflowKeepsCursor(t *testing.T) {
	grid := NewGrid(24, 80)
	grid.Feed([]byte("hello"))
	row, col := grid.Cursor()
	assert.Equal(t, 0, row)
	assert.Equal(t, 5, col)

	grid.Reflow(24, 3)
	row, col = grid.Cursor()
	assert.Equal(t, 1, row)
	assert.Equal(t, 2, col)
	assert.Equal(t, "hel\nlo", grid.Text())
}

func TestGridReflowShowsScrollback(t *testing.T) {
	grid := NewGrid(2, 4)
	grid.Feed([]byte("a\nb\nc"))
	assert.Equal(t, "b\nc", grid.Text())

	grid.Reflow(5, 4)
	assert.Equal(t, "a\nb\nc", grid.Text())
}

func TestGridSGRColor(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("\x1b[31mX"))
	_, _, _, _, cells := grid.Snapshot()
	require.NotEmpty(t, cells)
	assert.Equal(t, "X", cells[0].Rune)
	assert.Equal(t, 2, cells[0].FG)
}

func TestGridUnknownPrivateModeIsIgnored(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("\x1b[?9999hhi"))
	assert.Equal(t, "hi", grid.Text())
}

func TestGridTruecolorDoesNotStickRed(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("\x1b[48;2;197;52;41mY"))
	_, _, _, _, cells := grid.Snapshot()
	require.NotEmpty(t, cells)
	assert.Equal(t, "Y", cells[0].Rune)
	assert.Equal(t, ColorRGB, cells[0].BGMode)
	assert.Equal(t, uint32(197<<16|52<<8|41), cells[0].BGRGB)
	assert.Equal(t, 0, cells[0].BG)

	grid.Feed([]byte("\x1b[0;48:2::1:2:3mX"))
	_, _, _, _, cells = grid.Snapshot()
	assert.Equal(t, "X", cells[1].Rune)
	assert.Equal(t, ColorRGB, cells[1].BGMode)
	assert.Equal(t, uint32(1<<16|2<<8|3), cells[1].BGRGB)
	assert.Equal(t, 0, cells[1].BG)
	assert.Equal(t, 0, cells[1].FG)
}

func TestGridIndexedColor(t *testing.T) {
	grid := NewGrid(1, 4)
	grid.Feed([]byte("\x1b[38:5:196mZ"))
	_, _, _, _, cells := grid.Snapshot()
	require.NotEmpty(t, cells)
	assert.Equal(t, ColorIndex, cells[0].FGMode)
	assert.Equal(t, 196, cells[0].FGIndex)
	assert.Equal(t, 0, cells[0].FG)
}

func TestGridAltScreenRestores(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("ab\x1b[?1049h"))
	assert.Equal(t, "", grid.Text())
	grid.Feed([]byte("cd"))
	assert.Equal(t, "cd", grid.Text())
	grid.Feed([]byte("\x1b[?1049l"))
	assert.Equal(t, "ab", grid.Text())
}

func TestGridReverseAndScrollRegion(t *testing.T) {
	grid := NewGrid(4, 4)
	grid.Feed([]byte("\x1b[7mA"))
	_, _, _, _, cells := grid.Snapshot()
	assert.True(t, cells[0].Reverse)

	grid.Feed([]byte("\x1b[0m\x1b[1;2r\x1b[2;1Hb\nb"))
	assert.Equal(t, "b\nb", grid.Text())
}
