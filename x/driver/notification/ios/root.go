// Package ios posts a local notification.
// The app process uses UserNotifications. A host directory still exchanges request.json.
package ios

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_ios" }
func (factory) Name() string { return "iOS notification" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return iosbox.Require()
}

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Notify(ctx context.Context, n notification.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := iosbox.Call(ctx, iosbox.Request{
		Op:      iosbox.OpNotify,
		Title:   n.Title,
		Message: n.Message,
		Urgency: n.Urgency,
		ID:      n.ID,
	})
	if err != nil {
		return err
	}
	status, _, err := iosbox.Result(raw)
	if err != nil {
		return err
	}
	if status != askwire.StatusOK {
		return fmt.Errorf("%w: notification", driver.ErrUnavailable)
	}
	return nil
}

func init() { driver.Register[notification.Driver](factory{}) }

var (
	_ driver.DriverFactory[notification.Driver] = factory{}
	_ driver.Weighter                           = factory{}
)
