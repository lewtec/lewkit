package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrayUsage(t *testing.T) {
	text, err := cmd.Usage[Tray]("lewkit experiments tray")
	require.NoError(t, err)
	assert.Contains(t, text, "--icon")
	assert.Contains(t, text, "--name")
	assert.Contains(t, text, "Quit")
}

func TestTrayDefaultIcon(t *testing.T) {
	icon, err := (&Tray{}).loadIcon()
	require.NoError(t, err)
	require.NotNil(t, icon.Image)
	require.Equal(t, 32, icon.Image.Bounds().Dx())
}
