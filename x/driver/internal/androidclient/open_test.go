package androidclient

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
)

func TestOpenNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		test.Skip(t)
	}
	_, err := Open(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestInteractiveNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		test.Skip(t)
	}
	err := Interactive(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
