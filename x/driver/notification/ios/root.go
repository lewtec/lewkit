// Package ios posts a local notification through the packaged iOS host.
package ios

import (
	"context"
	"fmt"
	"runtime"

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
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if !iosbox.Available() {
		return fmt.Errorf("%w: no ios host", driver.ErrIncompatible)
	}
	return nil
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
