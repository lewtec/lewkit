package experiments

import (
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

func TestDoctorRuns(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[Command]](t, "doctor")
	require.NoError(t, app.Run(t.Context()))
}
