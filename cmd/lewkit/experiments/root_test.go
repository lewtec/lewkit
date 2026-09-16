package experiments

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoBarePrintsUsage(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[Command]](t, "demo")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "showcase the taskgroup executor")
	assert.Contains(t, got, "tasks")
	assert.Contains(t, got, "plain")
	assert.NotContains(t, got, "simulated 503")
}

func TestDemoUsage(t *testing.T) {
	text, err := cmd.Usage[Demo]("lewkit experiments demo")
	require.NoError(t, err)
	assert.Contains(t, text, "tasks")
	assert.Contains(t, text, "plain")
	assert.Contains(t, text, "nested")
	assert.Contains(t, text, "loop")
	assert.Contains(t, text, "map")
	assert.Contains(t, text, "many")
	assert.Contains(t, text, "tree")
	assert.Contains(t, text, "lines")
	assert.Contains(t, text, "rsync")
	assert.True(t, strings.Contains(text, "taskgroup") || strings.Contains(text, "showcase"))
}

func TestTreeDelayFlag(t *testing.T) {
	text, err := cmd.Usage[treeCmd]("lewkit experiments demo tree")
	require.NoError(t, err)
	assert.Contains(t, text, "--delay")
	assert.Contains(t, text, "350ms")
}
