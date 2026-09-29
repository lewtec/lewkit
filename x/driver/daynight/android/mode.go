package android

import "github.com/lewtec/lewkit/x/driver/daynight"

func modeFromUI(uiMode, mask, yes int) daynight.Mode {
	if mask != 0 && uiMode&mask == yes {
		return daynight.Dark
	}
	return daynight.Light
}
