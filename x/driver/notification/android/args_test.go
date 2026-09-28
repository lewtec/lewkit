package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/stretchr/testify/require"
)

func TestFactoryIdentity(t *testing.T) {
	t.Parallel()
	require.Equal(t, "notification_android", factory{}.ID())
	require.Equal(t, "Android", factory{}.Name())
	require.Equal(t, 80, factory{}.Weight())
}

func TestNotifyArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   notification.Notification
		want []string
	}{
		{
			name: "empty",
			want: []string{"0", "", "", "", ""},
		},
		{
			name: "status",
			in: notification.Notification{
				ID:          notification.StatusID,
				Title:       "Volume",
				Message:     "50%",
				Urgency:     "low",
				Progress:    0.5,
				HasProgress: true,
			},
			want: []string{"100", "Volume", "50%", "low", "50"},
		},
		{
			name: "progress clamps high",
			in:   notification.Notification{HasProgress: true, Progress: 1.5},
			want: []string{"0", "", "", "", "100"},
		},
		{
			name: "progress clamps low",
			in:   notification.Notification{HasProgress: true, Progress: -0.2},
			want: []string{"0", "", "", "", "0"},
		},
		{
			name: "progress ignored",
			in:   notification.Notification{Progress: 0.5},
			want: []string{"0", "", "", "", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, notifyArgs(tt.in))
		})
	}
}
