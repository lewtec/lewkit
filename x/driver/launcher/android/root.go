// Package android asks with an alert dialog on the foreground activity.
package android

import (
	"github.com/lewtec/lewkit/x/driver/androidask"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func init() {
	launcher.RegisterAsk("launcher_android", 80, "Android (Choose)", "Android (Prompt)", "Android (Confirm)", androidask.Platform, open)
}
