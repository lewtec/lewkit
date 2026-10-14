package android

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		test.Skip(t)
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
