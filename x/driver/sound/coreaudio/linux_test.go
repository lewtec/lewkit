//go:build linux

package coreaudio

import (
	"testing"

	lib "github.com/lewtec/lewkit/x/ffi/native/coreaudio"
	"github.com/stretchr/testify/require"
)

func TestUnavailableHere(t *testing.T) {
	require.Error(t, lib.Available())
	_, err := lib.Devices()
	require.Error(t, err)
}
