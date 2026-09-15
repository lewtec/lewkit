package main

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootUsage(t *testing.T) {
	text, err := cmd.Usage[cmd.App[root]]("lewkit")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(text, "Well planned primitives to be used in other projects.\n\nUsage:"))
	assert.Contains(t, text, "log verbosity (default: 0)")
	assert.Contains(t, text, "generate")
	assert.Contains(t, text, "--sentry-dsn")
	assert.Contains(t, text, "SENTRY_DSN")
}

func TestRootSentryDSN(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "https://public@example.com/1")
	assert.Equal(t, "https://public@example.com/1", app.Args.sentry.Value())
}

func TestRootSentryDSNDefault(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t)
	assert.Equal(t, "https://26fa6b84edbc334b77bf7f6e1d7d69bc@o4508616651505664.ingest.us.sentry.io/4512090764607488", app.Args.sentry.Value())
}

func TestGenerateDbUsage(t *testing.T) {
	text, err := cmd.Usage[generateCmd]("lewkit generate")
	require.NoError(t, err)
	assert.Contains(t, text, "db")
	assert.Contains(t, text, "shared Queries")
}
