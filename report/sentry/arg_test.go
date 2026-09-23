package sentry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgParse(t *testing.T) {
	t.Parallel()
	var a Arg
	require.NoError(t, a.Parse("https://public@example.com/1"))
	require.NotNil(t, a.Reporter)
}

func TestArgParseEmpty(t *testing.T) {
	t.Parallel()
	var a Arg
	require.NoError(t, a.Parse(""))
	require.NoError(t, a.Setup())
}

func TestArgParseBadDSN(t *testing.T) {
	t.Parallel()
	var a Arg
	require.Error(t, a.Parse("not-a-dsn"))
}
