package experiments

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	dsound "github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/audio_play/mem"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/sound"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibraryIngestAndQuery(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Ada", "Hits")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeTone(t, filepath.Join(dir, "song.wav"))
	writePNG(t, filepath.Join(dir, "cover.png"))

	lib, err := OpenLibrary(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, lib.Close()) })
	count, err := lib.Ingest(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	albums, err := lib.Albums(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, albums, 1)
	assert.Equal(t, "Hits", albums[0].Name)
	assert.Equal(t, "Ada", albums[0].Artist)
	assert.NotEmpty(t, albums[0].Cover)

	tracks, err := lib.Tracks(t.Context(), "Hits", "song")
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	assert.Equal(t, "song", tracks[0].Title)
	assert.Greater(t, tracks[0].Duration, time.Duration(0))

	none, err := lib.Tracks(t.Context(), "Hits", "missing")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestMusicViewPaintsLibrary(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Ada", "Hits")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	writeTone(t, filepath.Join(dir, "song.wav"))
	writePNG(t, filepath.Join(dir, "cover.png"))
	lib, err := OpenLibrary(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, lib.Close()) })
	_, err = lib.Ingest(t.Context(), root)
	require.NoError(t, err)
	model := newMusic(t.Context(), lib, newPlayer(nil))
	model.reload()
	require.NotNil(t, model.View())
	picture, err := gui.NewPicture()
	require.NoError(t, err)
	_, err = picture.Render(model.View(), gui.Size{Width: 900, Height: 700})
	require.NoError(t, err)
	require.NotEmpty(t, model.list.hits)
	require.NotNil(t, model.list.hits[0].box)
	var hit image.Point
	found := false
	for y := 699; y >= 0 && !found; y -= 2 {
		for x := 0; x < 900; x += 2 {
			point := image.Pt(x, y)
			if !model.list.hits[0].box.Contains(point) {
				continue
			}
			onAlbum := false
			for _, album := range model.shelf.hits {
				if album.box != nil && album.box.Contains(point) {
					onAlbum = true
					break
				}
			}
			if onAlbum {
				continue
			}
			hit = point
			found = true
			break
		}
	}
	require.True(t, found)
	next, cmd := model.Update(window.Pointer{Pos: hit, Button: 1, Pressed: true})
	require.NotNil(t, cmd)
	next, cmd = next.Update(cmd())
	require.Nil(t, cmd)
	painted := next.(*musicModel)
	assert.Equal(t, musicNow, painted.screen)
	assert.Equal(t, "song", painted.now.Title)
}

func TestMusicScaleGrowsRows(t *testing.T) {
	lib, err := OpenLibrary(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, lib.Close()) })
	small := newMusic(t.Context(), lib, newPlayer(nil))
	small.size = image.Pt(900, 700)
	small.tracks = []Track{{ID: 1, Title: "song", Artist: "Ada"}}
	small.View()
	require.NotEmpty(t, small.list.hits)
	big := newMusic(t.Context(), lib, newPlayer(nil))
	big.size = image.Pt(1800, 1400)
	big.tracks = small.tracks
	big.View()
	require.NotEmpty(t, big.list.hits)
	assert.Greater(t, big.list.hits[0].box.Width, small.list.hits[0].box.Width*1.5)
	assert.Greater(t, big.scale(), small.scale())
	_, cmd := big.Update(gui.TickMsg{Size: image.Pt(2400, 1600)})
	require.NotNil(t, cmd)
	assert.Equal(t, 2400, big.size.X)
}

func TestMusicFollowsLight(t *testing.T) {
	model := newMusic(t.Context(), nil, nil)
	next, cmd := model.Update(gui.ModeMsg{Mode: daynight.Light})
	require.Nil(t, cmd)
	view := next.(*musicModel).View().(*gui.Box)
	require.NotNil(t, view.Fill)
	assert.Equal(t, uint8(246), view.Fill.Red)
	assert.Equal(t, uint8(28), next.(*musicModel).paint.text.Red)
}

func TestChosenFolderIngests(t *testing.T) {
	root := t.TempDir()
	writeTone(t, filepath.Join(root, "song.wav"))
	lib, err := OpenLibrary(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, lib.Close()) })
	model := newMusic(t.Context(), lib, nil)
	model.View()
	require.NotNil(t, model.head.open)
	_, cmd := model.Update(chosen{paths: []string{root}})
	require.NotNil(t, cmd)
	got, ok := cmd().(ingested)
	require.True(t, ok)
	assert.Equal(t, 1, got.count)
	assert.NoError(t, got.err)
}

func TestPlayerReportsFrames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tone.wav")
	writeTone(t, path)
	var got *mem.Buffer
	play := newPlayer(func(_ context.Context, cfg dsound.Config) (io.WriteCloser, error) {
		buf, err := mem.Open(cfg)
		got = buf
		return buf, err
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go play.loop(ctx)
	play.Play(path, 0)
	require.Eventually(t, func() bool {
		return play.Played() == 800 && !play.Playing()
	}, time.Second, 5*time.Millisecond)
	require.NotNil(t, got)
	assert.Equal(t, 1600, len(got.PCM()))
	assert.Empty(t, play.Err())
}

func writeTone(t *testing.T, path string) {
	t.Helper()
	format := sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}
	pcm := bytes.Repeat([]byte{0, 1}, 800)
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, sound.WriteWAV(file, format, pcm))
	require.NoError(t, file.Close())
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Pix[0] = 255
	img.Pix[3] = 255
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, img))
	require.NoError(t, file.Close())
}
