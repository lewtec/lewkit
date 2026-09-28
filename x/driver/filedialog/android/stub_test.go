//go:build !android || !cgo

package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/stretchr/testify/require"
)

func TestIncompatibleWithoutVM(t *testing.T) {
	require.ErrorIs(t, factory{}.CheckCompatibility(t.Context()), driver.ErrIncompatible)
	_, err := factory{}.New(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	_, err = backend{}.Choose(t.Context(), filedialog.Request{})
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
