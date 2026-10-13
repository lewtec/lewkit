//go:build !android

package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJavaVM(t *testing.T) {
	err := JavaVM()
	require.ErrorIs(t, err, ErrIncompatible)
	require.EqualError(t, err, "component is incompatible: no Java VM")
}
