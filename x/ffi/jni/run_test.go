package jni

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunOnJava(t *testing.T) {
	var inline bool
	runOnJava(func() { inline = true })
	require.True(t, inline)
	require.Nil(t, currentRunner())

	var order []string
	setRunner(func(fn func()) {
		order = append(order, "runner")
		fn()
	})
	runOnJava(func() { order = append(order, "fn") })
	require.Equal(t, []string{"runner", "fn"}, order)
	require.NotNil(t, currentRunner())
}
