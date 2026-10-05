package win32

import (
	"strings"
	"unicode/utf16"
)

const (
	wsPopup      = 0x80000000
	wsCaption    = 0x00C00000
	wsSysMenu    = 0x00080000
	wsChild      = 0x40000000
	wsVisible    = 0x10000000
	wsBorder     = 0x00800000
	wsTabStop    = 0x00010000
	wsVScroll    = 0x00200000
	dsSetFont    = 0x40
	dsModalFrame = 0x80
	dsCenter     = 0x0800

	classButton = 0x0080
	classEdit   = 0x0081
	classStatic = 0x0082
	classList   = 0x0083

	idOK     = 1
	idCancel = 2
	idPrompt = 100
	idField  = 101
)

type control struct {
	class      uint16
	id         uint16
	style      uint32
	text       string
	x, y, w, h int16
}

type dialog struct {
	title    string
	w, h     int16
	controls []control
}

func promptDialog(title, text string) dialog {
	if title == "" {
		title = "Input"
	}
	return dialog{
		title: title,
		w:     220,
		h:     78,
		controls: []control{
			{class: classStatic, id: idPrompt, text: title, x: 8, y: 8, w: 204, h: 12, style: wsChild | wsVisible},
			{class: classEdit, id: idField, text: text, x: 8, y: 24, w: 204, h: 14, style: wsChild | wsVisible | wsBorder | wsTabStop | 0x80},
			{class: classButton, id: idOK, text: "OK", x: 112, y: 48, w: 46, h: 14, style: wsChild | wsVisible | wsTabStop | 1},
			{class: classButton, id: idCancel, text: "Cancel", x: 164, y: 48, w: 48, h: 14, style: wsChild | wsVisible | wsTabStop},
		},
	}
}

func chooseDialog(title string) dialog {
	if title == "" {
		title = "Choose"
	}
	return dialog{
		title: title,
		w:     220,
		h:     130,
		controls: []control{
			{class: classStatic, id: idPrompt, text: title, x: 8, y: 6, w: 204, h: 12, style: wsChild | wsVisible},
			{class: classList, id: idField, x: 8, y: 20, w: 204, h: 78, style: wsChild | wsVisible | wsBorder | wsVScroll | wsTabStop | 0x0101},
			{class: classButton, id: idOK, text: "OK", x: 112, y: 106, w: 46, h: 14, style: wsChild | wsVisible | wsTabStop | 1},
			{class: classButton, id: idCancel, text: "Cancel", x: 164, y: 106, w: 48, h: 14, style: wsChild | wsVisible | wsTabStop},
		},
	}
}

// bytes is a DLGTEMPLATE the dialog manager can show.
// Each control starts on a 4-byte boundary.
func (d dialog) bytes() []byte {
	var b []byte
	style := uint32(wsPopup | wsCaption | wsSysMenu | dsModalFrame | dsCenter | dsSetFont)
	b = appendU32(b, style)
	b = appendU32(b, 0)
	b = appendU16(b, uint16(len(d.controls)))
	b = appendU16(b, 0) // x
	b = appendU16(b, 0) // y
	b = appendU16(b, uint16(d.w))
	b = appendU16(b, uint16(d.h))
	b = appendU16(b, 0) // menu
	b = appendU16(b, 0) // class
	b = appendUTF16(b, d.title)
	b = appendU16(b, 9) // point size
	b = appendUTF16(b, "Segoe UI")
	for _, ctl := range d.controls {
		b = pad4(b)
		b = appendU32(b, ctl.style)
		b = appendU32(b, 0)
		b = appendU16(b, uint16(ctl.x))
		b = appendU16(b, uint16(ctl.y))
		b = appendU16(b, uint16(ctl.w))
		b = appendU16(b, uint16(ctl.h))
		b = appendU16(b, ctl.id)
		b = appendU16(b, 0xFFFF)
		b = appendU16(b, ctl.class)
		b = appendUTF16(b, ctl.text)
		b = appendU16(b, 0) // creation data
	}
	return b
}

func appendU16(b []byte, v uint16) []byte {
	return append(b, byte(v), byte(v>>8))
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendUTF16(b []byte, s string) []byte {
	// syscall.UTF16FromString exists only on windows. A NUL becomes an empty title.
	if strings.ContainsRune(s, 0) {
		s = ""
	}
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	for _, c := range u {
		b = appendU16(b, c)
	}
	return b
}

func pad4(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}
