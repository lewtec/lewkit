package metal

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestFactoryRejectsNonApple(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		test.Skip(t)
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
