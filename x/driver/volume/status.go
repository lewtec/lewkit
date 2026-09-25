package volume

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/notification"
)

// ShowStatus posts the default sink level as a progress notification.
func ShowStatus(ctx context.Context) error {
	level, err := GetVolume(ctx)
	if err != nil {
		return err
	}
	muted, err := GetMute(ctx)
	if err != nil {
		return err
	}
	sink, err := SinkName(ctx)
	if err != nil {
		return err
	}
	return notification.Notify(ctx, notification.Notification{
		ID:          notification.StatusID,
		Title:       "Volume",
		Message:     sink,
		Icon:        volumeIcon(level, muted),
		Progress:    level,
		HasProgress: true,
	})
}

func volumeIcon(level float64, muted bool) string {
	if muted || level == 0 {
		return "audio-volume-muted"
	}
	if level < .33 {
		return "audio-volume-low"
	}
	if level < .66 {
		return "audio-volume-medium"
	}
	return "audio-volume-high"
}
