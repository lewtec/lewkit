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
	assert.Contains(t, text, "compute")
}
