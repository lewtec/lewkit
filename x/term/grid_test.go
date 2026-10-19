package term

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
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

func TestGridRecordsMouseModes(t *testing.T) {
	grid := NewGrid(2, 8)
	grid.Feed([]byte("\x1b[?1000;1002;1003;1006;1004;2004;1007h"))
	want := InputMouse | InputMotion | InputAnyMotion | InputSGR | InputFocus | InputPaste | InputAltScroll
	assert.Equal(t, want, grid.Input())
	grid.Feed([]byte("\x1b[?1049h"))
	assert.Equal(t, want|InputAlt, grid.Input())
	grid.Feed([]byte("\x1b[?1000;1006l"))
	assert.Zero(t, grid.Input()&(InputMouse|InputSGR))
	assert.NotZero(t, grid.Input()&InputFocus)
	grid.Feed([]byte("\x1b[?9999hhi"))
	assert.Equal(t, "hi", grid.Text())
	grid.Feed([]byte("\x1bc"))
	assert.Zero(t, grid.Input())
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

func TestGridDrawsAKittyImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 16))
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, A: 255})
			img.SetNRGBA(x, y+8, color.NRGBA{B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	grid := NewGrid(3, 6)
	grid.Feed([]byte("\x1b_Ga=T,f=100,c=1,r=1;" + payload + "\x1b\\"))
	_, _, _, _, cells := grid.Snapshot()
	require.NotEmpty(t, cells)
	assert.Equal(t, "▀", cells[0].Rune)
	assert.Equal(t, ColorRGB, cells[0].FGMode)
	assert.Equal(t, ColorRGB, cells[0].BGMode)
	assert.Equal(t, uint32(200<<16), cells[0].FGRGB)
	assert.Equal(t, uint32(180), cells[0].BGRGB)

	grid.Feed([]byte("\x1b[1;2HZ"))
	_, _, _, _, cells = grid.Snapshot()
	assert.Equal(t, "▀", cells[0].Rune)
	assert.Equal(t, "Z", cells[1].Rune)

	grid.Feed([]byte("\x1b_Gnot an image\x1b\\Q"))
	assert.Contains(t, grid.Text(), "Q")
}

func TestGridOSCReportsDirectoryAndFacts(t *testing.T) {
	grid := NewGrid(4, 40)
	grid.Feed([]byte("hi\x1b]0;title\x07yo\x1b]7;file://host/tmp/room\x1b\\!\x1b]777;set;task;parse;more\x07\x1b]777;del;task\x1b\\"))
	assert.Equal(t, "hiyo!", grid.Text())
	reps := grid.takeReports()
	require.Len(t, reps, 3)
	assert.Equal(t, "/tmp/room", reps[0].cwd)
	assert.True(t, reps[1].set)
	assert.Equal(t, "task", reps[1].key)
	assert.Equal(t, "parse;more", reps[1].val)
	assert.True(t, reps[2].del)
	assert.Equal(t, "task", reps[2].key)
	assert.Empty(t, grid.takeReports())

	grid.Feed([]byte("\x1b]7;/var/plain\x07\x1b]1337;CurrentDir=/var/room\x07\x1b]777;cwd;/var/via\x07\x1b]777;set;bad key;nope\x07Z"))
	assert.Equal(t, "hiyo!Z", grid.Text())
	reps = grid.takeReports()
	require.Len(t, reps, 3)
	assert.Equal(t, "/var/plain", reps[0].cwd)
	assert.Equal(t, "/var/room", reps[1].cwd)
	assert.Equal(t, "/var/via", reps[2].cwd)

	grid.Feed([]byte("\x1bc"))
	assert.Empty(t, grid.takeReports())
	assert.Empty(t, grid.Text())
}
