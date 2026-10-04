// Package android posts a local notification through NotificationManager.
//
// An empty title is shown as Notification. ID replaces a previous alert.
// Urgency low, normal, and critical select the channel importance.
// The first post on API 33 asks for POST_NOTIFICATIONS.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

var _ driver.DriverFactory[notification.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() { driver.Register[notification.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "notification_android" }
func (factory) Name() string { return "Android notification" }
func (factory) Weight() int  { return 80 }
