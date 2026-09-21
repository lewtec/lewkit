package gui

import (
	"image"
	"slices"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"golang.org/x/image/font"
)

const (
	notePad   = 10
	noteInset = 10
)

// Notepad is an unsaved text editor. View builds a Box/Flex/Text tree.
// Nil Face uses [lewimage.Face].
type Notepad struct {
	Face font.Face

	size     image.Point
	body     []rune
	cursor   int
	caret    bool
	last     time.Duration
	bodyText *Text
}

// NewNotepad returns an unsaved editor. The error is always nil.
func NewNotepad() (*Notepad, error) {
	return &Notepad{
		size:   image.Pt(640, 480),
		body:   []rune("type here"),
		cursor: 9,
	}, nil
}

func (n *Notepad) Init() Cmd { return Tick() }

func (n *Notepad) Update(msg Msg) (Model, Cmd) {
	if n == nil {
		return n, nil
	}
	var cmd Cmd
	if t, ok := msg.(TickMsg); ok {
		n.last = t.Elapsed
		n.caret = t.Elapsed.Milliseconds()/400%2 == 0
		cmd = Every(t.Period)
	}
	switch e := msg.(type) {
	case window.Key:
		if e.Pressed {
			n.key(e)
		}
	case window.Pointer:
		if e.Button == 1 && e.Pressed && n.bodyText != nil {
			n.cursor = n.bodyText.indexAt(e.Pos)
			n.caret = true
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		n.size = size
	}
	return n, cmd
}

func (n *Notepad) View() Node {
	if n == nil {
		return nil
	}
	titleH := float32(lewimage.LineHeight(n.Face) + 10)
	n.bodyText = &Text{Value: string(n.body), Face: n.Face, Cursor: n.cursor, Caret: n.caret}
	return &Box{
		Padding: EdgeInsets{notePad, notePad, notePad, notePad},
		Fill:    &Color{32, 32, 38, 255},
		Radius:  8,
		Clip:    true,
		Child: &Flex{Axis: Vertical, Children: []FlexChild{
			{Child: &Box{
				Height:  titleH,
				Fill:    &Color{24, 24, 28, 255},
				Radius:  4,
				Padding: EdgeInsets{Left: noteInset, Top: 4},
				Child:   &Text{Value: "untitled", Face: n.Face},
			}},
			Expanded(&Box{
				Fill:    &Color{18, 18, 22, 255},
				Radius:  4,
				Padding: EdgeInsets{noteInset, noteInset, noteInset, noteInset},
				Child:   n.bodyText,
			}),
		}},
	}
}

func (n *Notepad) key(k window.Key) {
	if k.Mod&window.ModCtrl != 0 || k.Mod&window.ModSuper != 0 {
		return
	}
	switch {
	case n.backspace(k):
		n.deleteBehind()
	case n.enter(k):
		n.insert('\n')
	case n.left(k):
		if n.cursor > 0 {
			n.cursor--
		}
	case n.right(k):
		if n.cursor < len(n.body) {
			n.cursor++
		}
	case k.Rune >= 32 && k.Rune != 127:
		n.insert(k.Rune)
	}
}

func (n *Notepad) insert(r rune) {
	n.body = slices.Insert(n.body, n.cursor, r)
	n.cursor++
}

func (n *Notepad) deleteBehind() {
	if n.cursor > 0 {
		n.body = append(n.body[:n.cursor-1], n.body[n.cursor:]...)
		n.cursor--
	}
}

func (*Notepad) backspace(k window.Key) bool {
	return k.Rune == 8 || k.Rune == 127 || k.Code == 51 || k.Code == 8 || k.Code == 22
}

func (*Notepad) enter(k window.Key) bool {
	return k.Rune == '\r' || k.Rune == '\n' || k.Code == 36 || k.Code == 13 || k.Code == 24
}

func (*Notepad) left(k window.Key) bool {
	return k.Code == 123 || k.Code == 0x25 || k.Code == 113
}

func (*Notepad) right(k window.Key) bool {
	return k.Code == 124 || k.Code == 0x27 || k.Code == 114
}
