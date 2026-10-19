package term

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

const (
	mousePress = iota
	mouseRelease
	mouseWheel
	mouseMotion
)

// inputBytes is the terminal sequence for one key, in the shape an
// xterm-256color process reads.
func inputBytes(msg tea.KeyPressMsg) []byte {
	key := msg.Key()
	if key.Mod == tea.ModCtrl && key.Code >= 'a' && key.Code <= 'z' {
		return []byte{byte(key.Code - 'a' + 1)}
	}
	if key.Text != "" && key.Mod&^tea.ModShift == 0 {
		return []byte(key.Text)
	}
	mod := xtermMod(key.Mod)
	if seq, ok := specialBytes(keyBase(msg.Keystroke()), mod); ok {
		return seq
	}
	if key.Text != "" && key.Mod&(tea.ModCtrl|tea.ModSuper|tea.ModHyper) == 0 && key.Mod&(tea.ModAlt|tea.ModMeta) != 0 {
		return append([]byte{0x1b}, key.Text...)
	}
	code := key.Code
	if code == 0 && key.Text != "" {
		code = []rune(key.Text)[0]
	}
	if code > 0 && code != tea.KeyExtended {
		return csiU(code, mod)
	}
	return nil
}

func keyBase(stroke string) string {
	s := stroke
	for _, prefix := range []string{"ctrl+", "alt+", "shift+", "meta+", "hyper+", "super+"} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
		}
	}
	return s
}

func xtermMod(mod tea.KeyMod) int {
	n := 1
	if mod&tea.ModShift != 0 {
		n += 1
	}
	if mod&tea.ModAlt != 0 || mod&tea.ModMeta != 0 {
		n += 2
	}
	if mod&tea.ModCtrl != 0 {
		n += 4
	}
	if mod&tea.ModSuper != 0 {
		n += 8
	}
	if mod&tea.ModHyper != 0 {
		n += 16
	}
	return n
}

func specialBytes(base string, mod int) ([]byte, bool) {
	if mod < 1 {
		mod = 1
	}
	if base == "tab" && mod == 2 {
		return []byte("\x1b[Z"), true
	}
	switch base {
	case "enter":
		if mod == 1 {
			return []byte{'\r'}, true
		}
		return csiU(13, mod), true
	case "backspace":
		if mod == 1 {
			return []byte{0x7f}, true
		}
		return csiU(127, mod), true
	case "tab":
		if mod == 1 {
			return []byte{'\t'}, true
		}
		return csiU(9, mod), true
	case "esc":
		if mod == 1 {
			return []byte{0x1b}, true
		}
		return csiU(27, mod), true
	case "space":
		if mod == 1 {
			return []byte{' '}, true
		}
		return csiU(' ', mod), true
	case "up":
		return arrow('A', mod), true
	case "down":
		return arrow('B', mod), true
	case "right":
		return arrow('C', mod), true
	case "left":
		return arrow('D', mod), true
	case "home":
		return cursorLetter('H', mod), true
	case "end":
		return cursorLetter('F', mod), true
	case "insert":
		return tilde(2, mod), true
	case "delete":
		return tilde(3, mod), true
	case "pgup":
		return tilde(5, mod), true
	case "pgdown":
		return tilde(6, mod), true
	case "f1":
		return function(0, 'P', mod), true
	case "f2":
		return function(0, 'Q', mod), true
	case "f3":
		return function(0, 'R', mod), true
	case "f4":
		return function(0, 'S', mod), true
	case "f5":
		return tilde(15, mod), true
	case "f6":
		return tilde(17, mod), true
	case "f7":
		return tilde(18, mod), true
	case "f8":
		return tilde(19, mod), true
	case "f9":
		return tilde(20, mod), true
	case "f10":
		return tilde(21, mod), true
	case "f11":
		return tilde(23, mod), true
	case "f12":
		return tilde(24, mod), true
	default:
		return nil, false
	}
}

func arrow(final byte, mod int) []byte {
	if mod == 1 {
		return []byte{0x1b, '[', final}
	}
	return fmt.Appendf(nil, "\x1b[1;%d%c", mod, final)
}

func cursorLetter(final byte, mod int) []byte {
	if mod == 1 {
		return []byte{0x1b, '[', final}
	}
	return fmt.Appendf(nil, "\x1b[1;%d%c", mod, final)
}

func tilde(code, mod int) []byte {
	if mod == 1 {
		return fmt.Appendf(nil, "\x1b[%d~", code)
	}
	return fmt.Appendf(nil, "\x1b[%d;%d~", code, mod)
}

func function(code int, ss3 byte, mod int) []byte {
	if code == 0 {
		if mod == 1 {
			return []byte{0x1b, 'O', ss3}
		}
		return fmt.Appendf(nil, "\x1b[1;%d%c", mod, ss3)
	}
	return tilde(code, mod)
}

func csiU(code rune, mod int) []byte {
	if mod <= 1 {
		return fmt.Appendf(nil, "\x1b[%du", code)
	}
	return fmt.Appendf(nil, "\x1b[%d;%du", code, mod)
}

func pasteBytes(text string, bits uint32) []byte {
	if text == "" {
		return nil
	}
	if bits&InputPaste == 0 {
		return []byte(text)
	}
	return []byte("\x1b[200~" + text + "\x1b[201~")
}

func focusBytes(gain bool, bits uint32) []byte {
	if bits&InputFocus == 0 {
		return nil
	}
	if gain {
		return []byte("\x1b[I")
	}
	return []byte("\x1b[O")
}

func encodeMouse(ms tea.Mouse, kind, col, row int, bits uint32) []byte {
	if col < 1 {
		col = 1
	}
	if row < 1 {
		row = 1
	}
	if kind == mouseWheel || wheelButton(ms.Button) {
		if mouseTracking(bits) {
			code, ok := buttonCode(ms.Button, false)
			if !ok {
				return nil
			}
			return writeMouse(code+mouseMods(ms.Mod), col, row, false, bits)
		}
		if bits&(InputAlt|InputAltScroll) == 0 {
			return nil
		}
		switch ms.Button {
		case tea.MouseWheelUp:
			return []byte("\x1b[A")
		case tea.MouseWheelDown:
			return []byte("\x1b[B")
		case tea.MouseWheelLeft:
			return []byte("\x1b[D")
		case tea.MouseWheelRight:
			return []byte("\x1b[C")
		default:
			return nil
		}
	}
	if !mouseTracking(bits) {
		return nil
	}
	if kind == mouseMotion && !wantMotion(bits, ms.Button) {
		return nil
	}
	code, ok := buttonCode(ms.Button, kind == mouseMotion)
	if !ok {
		return nil
	}
	if kind == mouseRelease && bits&InputSGR == 0 {
		code = 3
	}
	return writeMouse(code+mouseMods(ms.Mod), col, row, kind == mouseRelease, bits)
}

func mouseTracking(bits uint32) bool {
	return bits&(InputMouse|InputMotion|InputAnyMotion) != 0
}

func wantMotion(bits uint32, btn tea.MouseButton) bool {
	if bits&InputAnyMotion != 0 {
		return true
	}
	return bits&InputMotion != 0 && btn != tea.MouseNone && !wheelButton(btn)
}

func wheelButton(btn tea.MouseButton) bool {
	switch btn {
	case tea.MouseWheelUp, tea.MouseWheelDown, tea.MouseWheelLeft, tea.MouseWheelRight:
		return true
	default:
		return false
	}
}

func buttonCode(btn tea.MouseButton, motion bool) (int, bool) {
	code := 0
	switch btn {
	case tea.MouseLeft:
		code = 0
	case tea.MouseMiddle:
		code = 1
	case tea.MouseRight:
		code = 2
	case tea.MouseNone:
		if !motion {
			return 0, false
		}
		code = 3
	case tea.MouseWheelUp:
		code = 64
	case tea.MouseWheelDown:
		code = 65
	case tea.MouseWheelLeft:
		code = 66
	case tea.MouseWheelRight:
		code = 67
	case tea.MouseBackward:
		code = 128
	case tea.MouseForward:
		code = 129
	default:
		return 0, false
	}
	if motion {
		code += 32
	}
	return code, true
}

func mouseMods(mod tea.KeyMod) int {
	n := 0
	if mod&tea.ModShift != 0 {
		n += 4
	}
	if mod&tea.ModAlt != 0 || mod&tea.ModMeta != 0 {
		n += 8
	}
	if mod&tea.ModCtrl != 0 {
		n += 16
	}
	return n
}

func writeMouse(code, col, row int, release bool, bits uint32) []byte {
	if bits&InputSGR != 0 {
		mark := byte('M')
		if release {
			mark = 'm'
		}
		return fmt.Appendf(nil, "\x1b[<%d;%d;%d%c", code, col, row, mark)
	}
	if col > 223 {
		col = 223
	}
	if row > 223 {
		row = 223
	}
	if code < 0 {
		code = 0
	}
	if code > 223 {
		code = 223
	}
	return []byte{0x1b, '[', 'M', byte(code + 32), byte(col + 32), byte(row + 32)}
}
