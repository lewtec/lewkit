package cocoa

import "github.com/lewtec/lewkit/x/driver/colorscheme"

// fromInterfaceStyle maps AppleInterfaceStyle. "Dark" is dark. Missing is light.
func fromInterfaceStyle(value string) colorscheme.Scheme {
	if value == "Dark" {
		return colorscheme.Dark
	}
	return colorscheme.Light
}
