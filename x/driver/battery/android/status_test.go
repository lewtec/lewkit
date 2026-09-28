package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

func TestStatusFromHost(t *testing.T) {
	tests := []struct {
		name string
		text string
		want battery.Status
		err  error
	}{
		{name: "charging", text: "Charging\n", want: battery.Charging},
		{name: "discharging", text: "Discharging", want: battery.Discharging},
		{name: "full", text: "Full", want: battery.Full},
		{name: "other", text: "Not charging\n", want: battery.Status("Not charging")},
		{name: "empty", text: "", want: battery.Unknown, err: battery.ErrNoBattery},
		{name: "blank", text: " \n", want: battery.Unknown, err: battery.ErrNoBattery},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := statusFromHost(tt.text)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}
