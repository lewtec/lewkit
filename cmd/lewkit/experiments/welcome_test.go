package experiments

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWelcomeUsage(t *testing.T) {
	text, err := cmd.Usage[welcomeCmd]("lewkit experiments welcome")
	require.NoError(t, err)
	assert.Contains(t, text, "pick a folder")
	assert.Contains(t, text, "--logo")
	assert.Contains(t, text, "--accent")
}

func TestLoadLogo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mark.png")
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, image.NewNRGBA(image.Rect(0, 0, 2, 2))))
	require.NoError(t, file.Close())
	img, err := loadLogo(path)
	require.NoError(t, err)
	assert.Equal(t, 2, img.Bounds().Dx())
}
