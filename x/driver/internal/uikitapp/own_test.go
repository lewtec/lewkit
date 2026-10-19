package uikitapp

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestOwnsUI(t *testing.T) {
	if runtime.GOOS == "ios" {
		test.Skip(t)
	}
	require.False(t, OwnsUI())
	require.ErrorIs(t, RequireProcess(), driver.ErrIncompatible)
}
