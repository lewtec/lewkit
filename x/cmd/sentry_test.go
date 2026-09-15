package cmd

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentryArgParse(t *testing.T) {
	app := ParseOK[App[None]](t, "--sentry-dsn", "https://public@example.com/1")
	assert.Equal(t, "https://public@example.com/1", app.sentry.Value())
}

func TestSentryArgEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	app := ParseOK[App[None]](t)
	assert.Equal(t, "https://env@example.com/1", app.sentry.Value())
}

func TestSentryArgFlagWinsEnv(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://env@example.com/1")
	app := ParseOK[App[None]](t, "--sentry-dsn", "https://flag@example.com/1")
	assert.Equal(t, "https://flag@example.com/1", app.sentry.Value())
}

func TestSentryArgSetupEmpty(t *testing.T) {
	var s SentryArg
	require.NoError(t, s.Setup())
}

func TestSentryArgSetupBadDSN(t *testing.T) {
	var s SentryArg
	require.NoError(t, s.Parse("not-a-dsn"))
	assert.Error(t, s.Setup())
}

func TestAppRunBadSentryDSN(t *testing.T) {
	test.RestoreSlog(t)
	app := ParseOK[App[None]](t, "--sentry-dsn", "not-a-dsn")
	assert.Error(t, app.Run(t.Context()))
}
