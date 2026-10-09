//go:build linux && !android

package wayland

import (
	"encoding/binary"
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/sys/unix"
)

func (w *wlwin) onSeat(id uint32, op uint16, payload []byte) {
	if op != 0 || len(payload) < 4 {
		return
	}
	caps := binary.LittleEndian.Uint32(payload)
	if caps&1 != 0 && w.pointerID == 0 {
		w.pointerID = w.conn.alloc(w.onPointer)
		_ = w.conn.send(id, 0, pack(w.pointerID))
	}
	if caps&2 != 0 && w.keyboardID == 0 {
		w.keyboardID = w.conn.alloc(w.onKeyboard)
		_ = w.conn.send(id, 1, pack(w.keyboardID))
	}
}

func (w *wlwin) onPointer(op uint16, payload []byte) {
	switch op {
	case 0: // enter
		_, payload = takeU32(payload)
		_, payload = takeU32(payload)
		x, payload := takeI32(payload)
		y, _ := takeI32(payload)
		w.point = image.Pt(fixedToInt(x), fixedToInt(y))
		w.Buffer.Emit(window.Pointer{Pos: w.point, Buttons: w.buttons})
	case 2: // motion
		_, payload = takeU32(payload)
		x, payload := takeI32(payload)
		y, _ := takeI32(payload)
		w.point = image.Pt(fixedToInt(x), fixedToInt(y))
		w.Buffer.Emit(window.Pointer{Pos: w.point, Buttons: w.buttons})
	case 3: // button
		_, payload = takeU32(payload)
		_, payload = takeU32(payload)
		code, payload := takeU32(payload)
		state, _ := takeU32(payload)
		number, mask := pointerButton(code)
		if number == 0 {
			return
		}
		pressed := state == 1
		if pressed {
			w.buttons |= mask
		} else {
			w.buttons &^= mask
		}
		w.Buffer.Emit(window.Pointer{Pos: w.point, Button: number, Pressed: pressed, Buttons: w.buttons})
	case 4: // axis
		_, payload = takeU32(payload)
		axis, payload := takeU32(payload)
		value, _ := takeI32(payload)
		delta := int(value >> 8)
		if delta == 0 && value != 0 {
			if value > 0 {
				delta = 1
			} else {
				delta = -1
			}
		}
		step := image.Point{}
		if axis == 0 {
			step.Y = delta
		} else if axis == 1 {
			step.X = delta
		}
		if step != (image.Point{}) {
			w.Buffer.Emit(window.Scroll{Pos: w.point, Delta: step})
		}
	}
}

func (w *wlwin) onKeyboard(op uint16, payload []byte) {
	switch op {
	case 0: // keymap
		if fd := w.conn.takeFD(); fd >= 0 {
			_ = unix.Close(fd)
		}
	case 3: // key
		_, payload = takeU32(payload)
		_, payload = takeU32(payload)
		code, payload := takeU32(payload)
		state, _ := takeU32(payload)
		shift := w.mods&window.ModShift != 0
		w.Buffer.Emit(window.Key{
			Rune:    evdevRune(code, shift),
			Code:    code,
			Pressed: state == 1,
			Mod:     w.mods,
		})
	case 4: // modifiers
		_, payload = takeU32(payload)
		depressed, _ := takeU32(payload)
		w.mods = waylandMod(depressed)
	}
}

func fixedToInt(v int32) int { return int(v >> 8) }

// pointerButton converts an evdev button to the X11 button number and mask.
// 1 is left, 2 is middle, 3 is right.
func pointerButton(code uint32) (int, int) {
	switch code {
	case 0x110:
		return 1, window.ButtonLeft
	case 0x112:
		return 2, window.ButtonMiddle
	case 0x111:
		return 3, window.ButtonRight
	default:
		return 0, 0
	}
}
