package media

import "github.com/lewtec/lewkit/x/driver/notification"

// StatusNotification is the progress alert for meta.
// An empty title returns false. Icon is left empty. The caller posts the alert
// and may set Icon from GetArtCachePath.
func StatusNotification(meta *Metadata) (notification.Notification, bool) {
	if meta == nil || meta.Title == "" {
		return notification.Notification{}, false
	}
	progress := 0.0
	if meta.Length > 0 {
		progress = float64(meta.Position) / float64(meta.Length)
	}
	title := meta.Title
	message := meta.Artist
	if message == "" {
		message = "Unknown Artist"
	}
	return notification.Notification{
		ID:          notification.StatusID,
		Title:       title,
		Message:     message,
		Progress:    progress,
		HasProgress: true,
	}, true
}
