package wkwebview

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestCheckCompatibility(t *testing.T) {
	err := factory{}.CheckCompatibility(t.Context())
	if runtime.GOOS != "darwin" {
		require.ErrorIs(t, err, driver.ErrIncompatible)
	}
}
