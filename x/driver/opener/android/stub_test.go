//go:build !android || !cgo

package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestStubIncompatible(t *testing.T) {
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	_, err = factory{}.New(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)

	var found bool
	for _, iface := range driver.Doctor(t.Context()) {
		for _, item := range iface.Drivers {
			if item.ID != "opener_android" {
				continue
			}
			found = true
			require.False(t, item.Available)
			require.ErrorIs(t, item.Error, driver.ErrIncompatible)
		}
	}
	require.True(t, found)
}
