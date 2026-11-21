package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ParseOK parses args as T and fatals if Parse returns an error.
func ParseOK[T any](t testing.TB, args ...string) T {
	t.Helper()
	got, err := Parse[T](args...)
	require.NoError(t, err)
	return got
}

// ParseErr parses args as T and fatals if Parse succeeds.
func ParseErr[T any](t testing.TB, args ...string) error {
	t.Helper()
	_, err := Parse[T](args...)
	require.Error(t, err)
	return err
}
