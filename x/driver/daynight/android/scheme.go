package android

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
)

func schemeFromHost(text string) (daynight.Mode, error) {
	switch strings.TrimSpace(text) {
	case "dark":
		return daynight.Dark, nil
	case "light":
		return daynight.Light, nil
	default:
		return daynight.Light, fmt.Errorf("%w: color scheme %q", driver.ErrUnavailable, text)
	}
}

// schemeFromBit maps the night bit the host sends. 1 is dark. 0 is light.
func schemeFromBit(mode int) (daynight.Mode, bool) {
	switch mode {
	case 1:
		return daynight.Dark, true
	case 0:
		return daynight.Light, true
	default:
		return daynight.Light, false
	}
}
