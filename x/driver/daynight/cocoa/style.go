package cocoa

import "github.com/lewtec/lewkit/x/driver/daynight"

// fromInterfaceStyle maps AppleInterfaceStyle. "Dark" is dark. Missing is light.
func fromInterfaceStyle(value string) daynight.Mode {
	if value == "Dark" {
		return daynight.Dark
	}
	return daynight.Light
}
