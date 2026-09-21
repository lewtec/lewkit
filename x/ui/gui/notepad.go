package gui

import (
	"image"
	"slices"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ndarray"
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

	picture  *Picture
	size     image.Point
	body     []rune
	cursor   int
	caret    bool
	last     time.Duration
	bodyText *Text
}

// NewNotepad compiles the Picture kernel used by View.
func NewNotepad() (*Notepad, error) {
	p, err := NewPicture(4)
	if err != nil {
		return nil, err
	}
	return &Notepad{
		picture: p,
		size:    image.Pt(640, 480),
		body:    []rune("type here"),
		cursor:  9,
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
			box := Rect{n.bodyText.origin.X, n.bodyText.origin.Y, n.bodyText.size.Width, n.bodyText.size.Height}
			n.cursor = textIndex(box, e.Pos, n.body, n.Face)
			n.caret = true
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		n.size = size
	}
	return n, cmd
}

func (n *Notepad) frameSig() uint64 {
	if n == nil {
		return 0
	}
	return n.picture.frameSig()
}

func (n *Notepad) View() *ndarray.Tensor[uint8] {
	if n == nil || n.picture == nil {
		return nil
	}
	sz := Size{float32(n.size.X), float32(n.size.Y)}
	t, err := n.picture.Render(n.tree(), sz)
	if err != nil {
		return n.picture.pixels
	}
	return t
}

func (n *Notepad) tree() Node {
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
	case keyBackspace(k):
		if n.cursor > 0 {
			n.body = append(n.body[:n.cursor-1], n.body[n.cursor:]...)
			n.cursor--
		}
	case keyReturn(k):
		n.insert('\n')
	case keyLeft(k):
		if n.cursor > 0 {
			n.cursor--
		}
	case keyRight(k):
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

func keyBackspace(k window.Key) bool {
	return k.Rune == 8 || k.Rune == 127 || k.Code == 51 || k.Code == 8 || k.Code == 22
}

func keyReturn(k window.Key) bool {
	return k.Rune == '\r' || k.Rune == '\n' || k.Code == 36 || k.Code == 13 || k.Code == 24
}

func keyLeft(k window.Key) bool {
	return k.Code == 123 || k.Code == 0x25 || k.Code == 113
}

func keyRight(k window.Key) bool {
	return k.Code == 124 || k.Code == 0x27 || k.Code == 114
}
