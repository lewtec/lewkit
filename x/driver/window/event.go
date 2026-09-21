package window

import "image"

// Event is a window notification. Subscribe to receive them.
type Event interface {
	windowEvent()
}

// Resize is the client size after a change.
type Resize struct {
	Size image.Point
}

func (Resize) windowEvent() {}

// Close means the window is gone.
type Close struct{}

func (Close) windowEvent() {}

// Expose means the host needs the frame shown again.
type Expose struct{}

func (Expose) windowEvent() {}

// Pointer buttons. Bit 0 is left, 1 is right, 2 is middle.
const (
	ButtonLeft   = 1 << 0
	ButtonRight  = 1 << 1
	ButtonMiddle = 1 << 2
)

// Pointer is a mouse sample in client pixels, same origin as Frame.
// Button is 1 left, 2 right, 3 middle; 0 means move. Pressed is the
// edge for that Button.
type Pointer struct {
	Pos     image.Point
	Button  int
	Pressed bool
	Buttons int
}

func (Pointer) windowEvent() {}

// Scroll is a wheel or trackpad step. +Y is down in Frame coordinates.
type Scroll struct {
	Pos   image.Point
	Delta image.Point
}

func (Scroll) windowEvent() {}

// Key is a keyboard sample. Rune is 0 when the event is not a character.
type Key struct {
	Rune    rune
	Code    uint32
	Pressed bool
	Repeat  bool
	Mod     Modifier
}

func (Key) windowEvent() {}

// Modifier bits for [Key].
type Modifier uint32

const (
	ModShift Modifier = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
)
