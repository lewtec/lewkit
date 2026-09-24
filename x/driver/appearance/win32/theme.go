package win32

import "github.com/lewtec/lewkit/x/driver/appearance"

// fromLightTheme maps AppsUseLightTheme. 0 is dark. Any other value, including a missing key, is light.
func fromLightTheme(value uint32) appearance.Scheme {
	if value == 0 {
		return appearance.Dark
	}
	return appearance.Light
}
