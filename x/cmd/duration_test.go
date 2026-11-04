package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationArg(t *testing.T) {
	type args struct {
		timeout DurationArg `long:"timeout"`
	}
	cases := []struct {
		name string
		in   string
		want time.Duration
		err  error
	}{
		{name: "millis", in: "300ms", want: 300 * time.Millisecond},
		{name: "seconds", in: "1s", want: time.Second},
		{name: "fractional hour", in: "1.5h", want: 90 * time.Minute},
		{name: "composite", in: "2h45m", want: 2*time.Hour + 45*time.Minute},
		{name: "zero", in: "0s", want: 0},
		{name: "empty", in: "", err: ErrInvalidArgument},
		{name: "bare number", in: "5", err: ErrInvalidArgument},
		{name: "garbage", in: "nope", err: ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[args]("--timeout", tc.in)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.timeout.Value())
		})
	}
}

func TestDurationDefault(t *testing.T) {
	type args struct {
		timeout DurationArg `long:"timeout" default:"30s"`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, got.timeout.Value())
}
