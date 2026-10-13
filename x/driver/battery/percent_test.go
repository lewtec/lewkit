package battery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPercent(t *testing.T) {
	tests := []struct {
		name string
		now  int64
		full int64
		want int
		err  error
	}{
		{name: "half", now: 5000, full: 10000, want: 50},
		{name: "empty", now: 0, full: 10000, want: 0},
		{name: "full", now: 10000, full: 10000, want: 100},
		{name: "over", now: 15000, full: 10000, want: 100},
		{name: "one third", now: 1, full: 3, want: 33},
		{name: "two thirds", now: 2, full: 3, want: 67},
		{name: "negative", now: -1, full: 10, err: ErrUnknownLevel},
		{name: "zero full", now: 1, full: 0, err: ErrUnknownLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Percent(tt.now, tt.full)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}
