package cocoa

import "github.com/lewtec/lewkit/x/driver/messagebox"

const (
	alertWarning       = 0
	alertInformational = 1
	alertCritical      = 2
)

func alertStyle(style string) int {
	switch messagebox.NormalizeStyle(style) {
	case messagebox.StyleWarning:
		return alertWarning
	case messagebox.StyleCritical:
		return alertCritical
	default:
		return alertInformational
	}
}
