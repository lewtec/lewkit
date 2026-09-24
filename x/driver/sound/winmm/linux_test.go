//go:build linux

package winmm

import (
	"testing"

	lib "github.com/lewtec/lewkit/x/ffi/native/winmm"
	"github.com/stretchr/testify/require"
)

func TestUnavailableHere(t *testing.T) {
	require.Error(t, lib.Available())
	_, err := lib.Devices()
	require.Error(t, err)
}
