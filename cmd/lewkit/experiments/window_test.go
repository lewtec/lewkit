package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowUsage(t *testing.T) {
	text, err := cmd.Usage[Window]("lewkit experiments window")
	require.NoError(t, err)
	assert.Contains(t, text, "triangle")
	assert.Contains(t, text, "one turn")
	assert.Contains(t, text, "perlin")
	assert.Contains(t, text, "compute")
	assert.Contains(t, text, "scroll")
	assert.Contains(t, text, "rounded")
	assert.Contains(t, text, "1024x1024")
	assert.Contains(t, text, "Vulkan")
	assert.Contains(t, text, "notepad")
	assert.Contains(t, text, "counter")
}
