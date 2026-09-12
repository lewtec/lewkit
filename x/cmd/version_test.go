package cmd

import (
	"testing"

	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/xtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type withVersion struct {
	version *VersionCmd
}

func TestVersionCmdRun(t *testing.T) {
	got := xtest.Stdout(t, func() {
		require.NoError(t, (VersionCmd{}).Run(t.Context()))
	})
	assert.Equal(t, release.Version()+"\n", got)
}

func TestVersionCmdOptIn(t *testing.T) {
	app, err := Parse[App[withVersion]]("version")
	require.NoError(t, err)
	require.NotNil(t, app.Args.version)
	got := xtest.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Equal(t, release.Version()+"\n", got)
}

func TestVersionCmdUsage(t *testing.T) {
	text, err := Usage[withVersion]("tool")
	require.NoError(t, err)
	assert.Contains(t, text, "version")
	assert.Contains(t, text, "print version")
}

func TestVersionCmdNotDefault(t *testing.T) {
	text, err := Usage[App[None]]("lewkit")
	require.NoError(t, err)
	assert.NotContains(t, text, "Commands:")
	assert.Contains(t, text, "--version")
}
