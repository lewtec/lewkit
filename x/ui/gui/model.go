package gui

import (
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Msg is an incoming event. Host messages are [window.Resize],
// [window.Expose], [window.Close], and [TickMsg].
type Msg any

// Cmd produces a follow-up [Msg]. Nil means no command.
type Cmd func() Msg

// Model is the bubbletea trio. View is a tensor, not a string.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() *ndarray.Tensor[uint8]
}

// TickMsg is one animation frame. Size is the window client size.
type TickMsg struct {
	Elapsed time.Duration
	Size    image.Point
}

func sizeOf(msg Msg) (image.Point, bool) {
	switch m := msg.(type) {
	case TickMsg:
		return m.Size, true
	case window.Resize:
		return m.Size, true
	default:
		return image.Point{}, false
	}
}
