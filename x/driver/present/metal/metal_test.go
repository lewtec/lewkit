package metal

import (
	"context"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestFactoryRejectsNonApple(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(context.Background())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
