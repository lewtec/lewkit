package dbus

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_dbus" }
func (factory) Name() string { return "DBus" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (notification.Driver, error) { return open(ctx) }

func init() { driver.Register[notification.Driver](factory{}) }
