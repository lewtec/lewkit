package main

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootUsage(t *testing.T) {
	text, err := cmd.Usage[cmd.App[root]]("lewkit")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(text, "Well planned primitives to be used in other projects.\n\nUsage:"))
	assert.Contains(t, text, "log verbosity (default: 0)")
	assert.Contains(t, text, "generate")
	assert.Contains(t, text, "disasm")
	assert.Contains(t, text, "experiments")
	assert.Contains(t, text, "completion")
	assert.Contains(t, text, "--sentry-dsn")
	assert.Contains(t, text, "SENTRY_DSN")
}

func TestRootSentryDSN(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "https://public@example.com/1")
	require.NotNil(t, app.Args.sentry.Reporter)
}

func TestCompletionLine(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "completion")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "complete -C")
	assert.Contains(t, got, "lewkit")
}

func TestGenerateDbUsage(t *testing.T) {
	text, err := cmd.Usage[generateCmd]("lewkit generate")
	require.NoError(t, err)
	assert.Contains(t, text, "db")
	assert.Contains(t, text, "shared Queries")
}

func TestGeneratePreludeUsage(t *testing.T) {
	text, err := cmd.Usage[generateCmd]("lewkit generate")
	require.NoError(t, err)
	assert.Contains(t, text, "prelude")
	assert.Contains(t, text, "blank-import")
}

func TestGenerateHelp(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "generate", "--help")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "generate code")
	assert.Contains(t, got, "db")
	assert.Contains(t, got, "prelude")
	assert.Contains(t, got, "shared Queries")
	assert.Contains(t, got, "--sentry-dsn")
}

func TestGenerateErrUsage(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "--sentry-dsn", "", "generate")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "generate code")
	assert.Contains(t, got, "db")
	assert.Contains(t, got, "prelude")
	assert.Contains(t, got, "--sentry-dsn")
}
