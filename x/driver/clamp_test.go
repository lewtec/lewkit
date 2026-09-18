package driver_test

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestClamp01(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want float64
	}{
		{in: -1, want: 0},
		{in: 0, want: 0},
		{in: 0.5, want: 0.5},
		{in: 1, want: 1},
		{in: 2, want: 1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.in), func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, driver.Clamp01(tt.in))
		})
	}
}
