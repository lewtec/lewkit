package volume

import "github.com/lewtec/lewkit/x/driver/notification"

// StatusNotification is the progress alert for a sink level.
// level is a fraction from 0 to 1. The caller posts it.
func StatusNotification(level float64, muted bool, sink string) notification.Notification {
	return notification.Notification{
		ID:          notification.StatusID,
		Title:       "Volume",
		Message:     sink,
		Icon:        volumeIcon(level, muted),
		Progress:    level,
		HasProgress: true,
	}
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
