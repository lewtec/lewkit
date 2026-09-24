package win32

import "github.com/lewtec/lewkit/x/driver/daynight"

// fromLightTheme maps AppsUseLightTheme. 0 is dark. Any other value, including a missing key, is light.
func fromLightTheme(value uint32) daynight.Mode {
	if value == 0 {
		return daynight.Dark
	}
	return daynight.Light
}
