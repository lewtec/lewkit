package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

func TestPercent(t *testing.T) {
	tests := []struct {
		name  string
		level int
		scale int
		want  int
		err   error
	}{
		{name: "half", level: 50, scale: 100, want: 50},
		{name: "zero", level: 0, scale: 100, want: 0},
		{name: "full", level: 100, scale: 100, want: 100},
		{name: "one third", level: 1, scale: 3, want: 33},
		{name: "two thirds", level: 2, scale: 3, want: 67},
		{name: "above scale", level: 150, scale: 100, want: 100},
		{name: "missing level", level: -1, scale: 100, err: battery.ErrUnknownLevel},
		{name: "bad scale", level: 10, scale: 0, err: battery.ErrUnknownLevel},
		{name: "negative scale", level: 10, scale: -1, err: battery.ErrUnknownLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := percent(tt.level, tt.scale)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStatusFrom(t *testing.T) {
	known := levels{unknown: 1, charging: 2, discharging: 3, notCharging: 4, full: 5}
	tests := []struct {
		name    string
		present bool
		code    int
		want    battery.Status
		err     error
	}{
		{name: "absent", present: false, code: 2, want: battery.Unknown, err: battery.ErrNoBattery},
		{name: "charging", present: true, code: 2, want: battery.Charging},
		{name: "discharging", present: true, code: 3, want: battery.Discharging},
		{name: "not charging", present: true, code: 4, want: battery.Status("Not charging")},
		{name: "full", present: true, code: 5, want: battery.Full},
		{name: "unknown", present: true, code: 1, want: battery.Unknown},
		{name: "other", present: true, code: 9, want: battery.Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := statusFrom(tt.present, tt.code, known)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}
