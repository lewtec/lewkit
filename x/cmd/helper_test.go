package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func parseOK[T any](t testing.TB, args ...string) T {
	t.Helper()
	got, err := Parse[T](args...)
	require.NoError(t, err)
	return got
}
