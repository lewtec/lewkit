package notify_send

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_notify_send" }
func (factory) Name() string { return "notify-send" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireBinary(ctx, "notify-send")
}

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

func init() { driver.Register[notification.Driver](factory{}) }
