package x11

import "github.com/jezek/xgb/xproto"

func (w *xwin) loadKeys() {
	if w == nil || w.conn == nil {
		return
	}
	setup := xproto.Setup(w.conn)
	if setup == nil || setup.MaxKeycode < setup.MinKeycode {
		return
	}
	count := byte(setup.MaxKeycode - setup.MinKeycode + 1)
	reply, err := xproto.GetKeyboardMapping(w.conn, setup.MinKeycode, count).Reply()
	if err != nil || reply == nil || reply.KeysymsPerKeycode < 1 {
		return
	}
	w.minKey = byte(setup.MinKeycode)
	w.symsPer = int(reply.KeysymsPerKeycode)
	w.keysyms = reply.Keysyms
}

func (w *xwin) runeOf(detail xproto.Keycode, state uint16) rune {
	return keysymRune(w.keysym(detail, state))
}

func (w *xwin) keysym(detail xproto.Keycode, state uint16) xproto.Keysym {
	if w == nil || w.symsPer < 1 || byte(detail) < w.minKey {
		return 0
	}
	index := int(byte(detail)-w.minKey) * w.symsPer
	if state&1 != 0 && w.symsPer > 1 {
		index++
	}
	if index < 0 || index >= len(w.keysyms) {
		return 0
	}
	return w.keysyms[index]
}

func keysymRune(sym xproto.Keysym) rune {
	switch sym {
	case 0xff08, 0xffff:
		return 8
	case 0xff0d, 0xff8d:
		return '\n'
	}
	if sym >= 0x20 && sym <= 0xff {
		return rune(sym)
	}
	return 0
}
