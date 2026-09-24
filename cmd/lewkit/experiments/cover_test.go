package experiments

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestID3APICBecomesCover(t *testing.T) {
	raw := tinyPNG(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	require.NoError(t, os.WriteFile(path, id3Tag(raw), 0o644))
	assert.Equal(t, raw, readEmbeddedCover(path))

	lib, err := OpenLibrary()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, lib.Close()) })
	saved := lib.trackCover(dir, path)
	require.NotEmpty(t, saved)
	model := newMusic(t.Context(), lib, newPlayer(nil))
	img := model.cover(saved)
	require.NotNil(t, img)
	assert.Equal(t, 2, img.Bounds().Dx())
}

func TestFolderCoverIgnoresCase(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Cover.JPG"), []byte{0xff, 0xd8}, 0o644))
	assert.Equal(t, filepath.Join(dir, "Cover.JPG"), folderCover(dir))
}

func id3Tag(image []byte) []byte {
	apic := []byte{0}
	apic = append(apic, []byte("image/png")...)
	apic = append(apic, 0, 3, 0)
	apic = append(apic, image...)
	frame := make([]byte, 10+len(apic))
	copy(frame[:4], "APIC")
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(apic)))
	copy(frame[10:], apic)
	size := len(frame)
	head := []byte{'I', 'D', '3', 3, 0, 0,
		byte(size >> 21), byte(size >> 14), byte(size >> 7), byte(size),
	}
	for i := 6; i < 10; i++ {
		head[i] &= 0x7f
	}
	return append(head, frame...)
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Pix[0], img.Pix[3] = 220, 255
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
