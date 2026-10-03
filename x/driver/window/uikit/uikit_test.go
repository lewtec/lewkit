package uikit

import (
	"context"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestFactoryRejectsNonIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(context.Background())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
