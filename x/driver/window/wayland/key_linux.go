//go:build linux && !android

package wayland

import "github.com/lewtec/lewkit/x/driver/window"

func waylandMod(mask uint32) window.Modifier {
	var mod window.Modifier
	if mask&0x1 != 0 {
		mod |= window.ModShift
	}
	if mask&0x4 != 0 {
		mod |= window.ModCtrl
	}
	if mask&0x8 != 0 {
		mod |= window.ModAlt
	}
	if mask&0x40 != 0 {
		mod |= window.ModSuper
	}
	return mod
}
