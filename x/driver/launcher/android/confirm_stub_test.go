//go:build !android || !cgo

package android

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestIncompatibleWithoutVM(t *testing.T) {
	t.Parallel()

	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	_, err = factory{}.New(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.Equal(t, "confirmer_android", factory{}.ID())
	require.Equal(t, "Android confirm", factory{}.Name())
	require.Equal(t, 80, factory{}.Weight())
}

func TestDoctorListsConfirm(t *testing.T) {
	report := driver.Doctor(t.Context())
	var found bool
	for _, iface := range report {
		for _, d := range iface.Drivers {
			if d.ID != "confirmer_android" {
				continue
			}
			found = true
			require.Equal(t, "Android confirm", d.Name)
			require.Equal(t, 80, d.Weight)
			require.False(t, d.Available)
			require.False(t, d.Selected)
			require.ErrorIs(t, d.Error, driver.ErrIncompatible)
		}
	}
	require.True(t, found)
}
