package brightness

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/notification"
)

// ShowStatus posts the backlight level as a progress notification.
func ShowStatus(ctx context.Context) error {
	status, err := Status(ctx)
	if err != nil {
		return err
	}
	return notification.Notify(ctx, notification.Notification{
		ID:          notification.StatusID,
		Title:       "Brightness",
		Message:     status.Name,
		Icon:        "display-brightness",
		Progress:    status.Brightness,
		HasProgress: true,
	})
}
