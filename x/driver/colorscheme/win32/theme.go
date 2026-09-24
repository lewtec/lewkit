package win32

import "github.com/lewtec/lewkit/x/driver/colorscheme"

// fromLightTheme maps AppsUseLightTheme. 0 is dark. Any other value, including a missing key, is light.
func fromLightTheme(value uint32) colorscheme.Scheme {
	if value == 0 {
		return colorscheme.Dark
	}
	return colorscheme.Light
}
