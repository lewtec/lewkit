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

func TestGridPrivateModeIsIgnored(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("\x1b[?1049hhi"))
	assert.Equal(t, "hi", grid.Text())
}
