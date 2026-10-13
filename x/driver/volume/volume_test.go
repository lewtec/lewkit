package volume

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestClamp01(t *testing.T) {
	require.Equal(t, 1.0, driver.Clamp01(1+step))
	require.Equal(t, 0.0, driver.Clamp01(0-step))
}
