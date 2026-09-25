package notify_send

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_notify_send" }
func (factory) Name() string { return "notify-send" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(context.Context) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("%w: notify-send not found", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

func init() { driver.Register[notification.Driver](factory{}) }
