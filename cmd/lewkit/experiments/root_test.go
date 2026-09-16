package experiments

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoUsage(t *testing.T) {
	text, err := cmd.Usage[Demo]("lewkit experiments demo")
	require.NoError(t, err)
	assert.Contains(t, text, "tasks")
	assert.Contains(t, text, "plain")
	assert.Contains(t, text, "nested")
	assert.Contains(t, text, "loop")
	assert.Contains(t, text, "map")
	assert.Contains(t, text, "many")
	assert.True(t, strings.Contains(text, "taskgroup") || strings.Contains(text, "showcase"))
}
