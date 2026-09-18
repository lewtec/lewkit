package experiments

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorUsage(t *testing.T) {
	text, err := cmd.Usage[Doctor]("lewkit experiments doctor")
	require.NoError(t, err)
	assert.Contains(t, text, "interface => implementation")
}

func TestDoctorPrintsTree(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[Command]](t, "doctor")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "window.Driver")
	assert.Contains(t, got, "=>")
	assert.Contains(t, got, "window_mem")
	assert.True(t, strings.Contains(got, "├ ") || strings.Contains(got, "└ "))
}
