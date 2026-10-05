// Package windows posts a local notification with Shell_NotifyIcon.
// The balloon uses the title and message. Low urgency is silent.
// Critical uses the error icon. ID replaces a previous balloon.
package windows

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_windows" }
func (factory) Name() string { return "Shell notification" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

type backend struct{}

func init() { driver.Register[notification.Driver](factory{}) }

var (
	_ driver.DriverFactory[notification.Driver] = factory{}
	_ driver.Weighter                           = factory{}
)
