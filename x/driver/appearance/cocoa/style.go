package cocoa

import "github.com/lewtec/lewkit/x/driver/appearance"

// fromInterfaceStyle maps AppleInterfaceStyle. "Dark" is dark. Missing is light.
func fromInterfaceStyle(value string) appearance.Scheme {
	if value == "Dark" {
		return appearance.Dark
	}
	return appearance.Light
}
