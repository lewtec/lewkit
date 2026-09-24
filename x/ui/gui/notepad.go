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
	Dirty
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

func (notepad *Notepad) Init() Cmd { return Tick() }

func (notepad *Notepad) Update(msg Msg) (Model, Cmd) {
	if notepad == nil {
		return notepad, nil
	}
	var cmd Cmd
	if tick, ok := msg.(TickMsg); ok {
		notepad.last = tick.Elapsed
		Set(notepad, &notepad.caret, tick.Elapsed.Milliseconds()/400%2 == 0)
		cmd = Every(tick.Period)
	}
	switch event := msg.(type) {
	case window.Key:
		if event.Pressed {
			notepad.key(event)
		}
	case window.Pointer:
		if event.Button == 1 && event.Pressed && notepad.bodyText != nil {
			Set(notepad, &notepad.cursor, notepad.bodyText.indexAt(event.Pos))
			Set(notepad, &notepad.caret, true)
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		Set(notepad, &notepad.size, size)
	}
	return notepad, cmd
}

func (notepad *Notepad) View() Node {
	if notepad == nil {
		return nil
	}
	titleHeight := float32(lewimage.LineHeight(notepad.Face) + 10)
	notepad.bodyText = &Text{Value: string(notepad.body), Face: notepad.Face, Cursor: notepad.cursor, Caret: notepad.caret}
	return &Box{
		Padding: EdgeInsets{notePad, notePad, notePad, notePad},
		Fill:    &Color{32, 32, 38, 255},
		Radius:  8,
		Clip:    true,
		Child: &Flex{Axis: Vertical, Children: []FlexChild{
			{Child: &Box{
				Height:  titleHeight,
				Fill:    &Color{24, 24, 28, 255},
				Radius:  4,
				Padding: EdgeInsets{Left: noteInset, Top: 4},
				Child:   &Text{Value: "untitled", Face: notepad.Face},
			}},
			Expanded(&Box{
				Fill:    &Color{18, 18, 22, 255},
				Radius:  4,
				Padding: EdgeInsets{noteInset, noteInset, noteInset, noteInset},
				Child:   notepad.bodyText,
			}),
		}},
	}
}

func (notepad *Notepad) key(key window.Key) {
	if key.Mod&window.ModCtrl != 0 || key.Mod&window.ModSuper != 0 {
		return
	}
	switch {
	case notepad.backspace(key):
		notepad.deleteBehind()
	case notepad.enter(key):
		notepad.insert('\n')
	case notepad.left(key):
		if notepad.cursor > 0 {
			Set(notepad, &notepad.cursor, notepad.cursor-1)
		}
	case notepad.right(key):
		if notepad.cursor < len(notepad.body) {
			Set(notepad, &notepad.cursor, notepad.cursor+1)
		}
	case key.Rune >= 32 && key.Rune != 127:
		notepad.insert(key.Rune)
	}
}

func (notepad *Notepad) insert(character rune) {
	notepad.body = slices.Insert(notepad.body, notepad.cursor, character)
	notepad.cursor++
	notepad.MarkDirty()
}

func (notepad *Notepad) deleteBehind() {
	if notepad.cursor > 0 {
		notepad.body = append(notepad.body[:notepad.cursor-1], notepad.body[notepad.cursor:]...)
		notepad.cursor--
		notepad.MarkDirty()
	}
}

func (*Notepad) backspace(key window.Key) bool {
	return key.Rune == 8 || key.Rune == 127 || key.Code == 51 || key.Code == 8 || key.Code == 22
}

func (*Notepad) enter(key window.Key) bool {
	return key.Rune == '\r' || key.Rune == '\n' || key.Code == 36 || key.Code == 13 || key.Code == 24
}

func (*Notepad) left(key window.Key) bool {
	return key.Code == 123 || key.Code == 0x25 || key.Code == 113
}

func (*Notepad) right(key window.Key) bool {
	return key.Code == 124 || key.Code == 0x27 || key.Code == 114
}
