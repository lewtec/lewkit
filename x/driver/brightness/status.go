package brightness

import "github.com/lewtec/lewkit/x/driver/notification"

// StatusNotification is the progress alert for a backlight.
// level is a fraction from 0 to 1. The caller posts it.
func StatusNotification(name string, level float64) notification.Notification {
	return notification.Notification{
		ID:          notification.StatusID,
		Title:       "Brightness",
		Message:     name,
		Icon:        "display-brightness",
		Progress:    level,
		HasProgress: true,
	}
}
