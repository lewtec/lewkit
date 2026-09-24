package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChooseUsage(t *testing.T) {
	text, err := cmd.Usage[FileDialog]("lewkit experiments filedialog")
	require.NoError(t, err)
	assert.Contains(t, text, "--folder")
	assert.Contains(t, text, "--save")
	assert.Contains(t, text, "--multiple")
	assert.Contains(t, text, "--filter")
	assert.Contains(t, text, "print each path")
}
