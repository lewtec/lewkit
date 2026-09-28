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
}
