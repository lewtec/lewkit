//go:build linux

package gtk

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/chooser"
	"github.com/stretchr/testify/require"
)

func TestGTKAvailableOrIncompatible(t *testing.T) {
	err := factory{}.CheckCompatibility(t.Context())
	t.Log(err)
	if err != nil {
		require.ErrorIs(t, err, driver.ErrIncompatible)
		return
	}
	got, err := factory{}.New(t.Context())
	require.NoError(t, err)
	_, ok := got.(chooser.Driver)
	require.True(t, ok)
}
