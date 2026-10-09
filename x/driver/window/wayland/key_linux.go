//go:build linux && !android

package wayland

import "github.com/lewtec/lewkit/x/driver/window"

// evdevRune maps a Linux evdev code to the unshifted and shifted rune.
// The Wayland keymap fd stays with the compositor; this is the pc105 set the
// pointer buttons do not need, and letters still arrive as characters.
func evdevRune(code uint32, shift bool) rune {
	row, ok := evdevKeys[code]
	if !ok {
		return 0
	}
	if shift {
		return row[1]
	}
	return row[0]
}

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

// evdevKeys is KEY_* from linux/input-event-codes.h for a US pc105 keyboard.
var evdevKeys = map[uint32][2]rune{
	1:  {0x1b, 0x1b},
	2:  {'1', '!'},
	3:  {'2', '@'},
	4:  {'3', '#'},
	5:  {'4', '$'},
	6:  {'5', '%'},
	7:  {'6', '^'},
	8:  {'7', '&'},
	9:  {'8', '*'},
	10: {'9', '('},
	11: {'0', ')'},
	12: {'-', '_'},
	13: {'=', '+'},
	14: {8, 8},
	15: {'\t', '\t'},
	16: {'q', 'Q'},
	17: {'w', 'W'},
	18: {'e', 'E'},
	19: {'r', 'R'},
	20: {'t', 'T'},
	21: {'y', 'Y'},
	22: {'u', 'U'},
	23: {'i', 'I'},
	24: {'o', 'O'},
	25: {'p', 'P'},
	26: {'[', '{'},
	27: {']', '}'},
	28: {'\n', '\n'},
	30: {'a', 'A'},
	31: {'s', 'S'},
	32: {'d', 'D'},
	33: {'f', 'F'},
	34: {'g', 'G'},
	35: {'h', 'H'},
	36: {'j', 'J'},
	37: {'k', 'K'},
	38: {'l', 'L'},
	39: {';', ':'},
	40: {'\'', '"'},
	41: {'`', '~'},
	43: {'\\', '|'},
	44: {'z', 'Z'},
	45: {'x', 'X'},
	46: {'c', 'C'},
	47: {'v', 'V'},
	48: {'b', 'B'},
	49: {'n', 'N'},
	50: {'m', 'M'},
	51: {',', '<'},
	52: {'.', '>'},
	53: {'/', '?'},
	57: {' ', ' '},
	78: {'+', '+'},
}
