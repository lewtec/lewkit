package common

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArch(t *testing.T) {
	unsupported := errors.New("unsupported GOARCH for linux")
	got, err := Arch("amd64", unsupported)
	require.NoError(t, err)
	assert.Equal(t, "amd64", got)

	got, err = Arch("arm64", unsupported)
	require.NoError(t, err)
	assert.Equal(t, "arm64", got)

	_, err = Arch("386", unsupported)
	require.ErrorIs(t, err, unsupported)
	assert.EqualError(t, err, "unsupported GOARCH for linux: 386")

	got, err = Arch("", unsupported)
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		require.NoError(t, err)
		assert.Equal(t, runtime.GOARCH, got)
		return
	}
	require.ErrorIs(t, err, unsupported)
}
