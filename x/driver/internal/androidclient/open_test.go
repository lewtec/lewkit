package androidclient

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestOpenNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip()
	}
	_, err := Open(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestInteractiveNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip()
	}
	err := Interactive(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
