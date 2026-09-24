package gui

import (
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
)

// Msg is an incoming event. Host messages are [window.Resize],
// [window.Expose], [window.Close], [window.Pointer], [window.Scroll],
// [window.Key], [TickMsg], and [SchemeMsg].
type Msg any

// Cmd produces a follow-up [Msg]. Nil means no command.
// Run runs Cmd asynchronously; the result is another message.
type Cmd func() Msg

// Tick immediately produces a [TickMsg]. Use from Init for the first frame.
func Tick() Cmd {
	return func() Msg { return TickMsg{} }
}

// Every produces a [TickMsg] one display period after this call.
// The wait is the time still left in that period, so a present that
// already took the frame does not sleep again. Models that animate
// return Every from Update after handling a tick.
func Every(duration time.Duration) Cmd {
	if duration <= 0 {
		duration = window.DefaultFramePeriod
	}
	deadline := time.Now().Add(duration)
	return func() Msg {
		if wait := time.Until(deadline); wait > 0 {
			time.Sleep(wait)
		}
		return TickMsg{}
	}
}

// Model is the bubbletea trio. View is a layout [Node], not a tensor.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Node
}

// TickMsg is the clock. Models request it with [Tick] and [Every]; Run
// does not tick on its own. Pointer, Scroll, Resize, and Key stay the
// messages [window] emits. Size is the window client size. Period is
// [window.Window.FramePeriod]. FPS is the smoothed frames-per-second
// since the previous tick.
type TickMsg struct {
	Elapsed time.Duration
	Size    image.Point
	FPS     float64
	Period  time.Duration
}

func sizeOf(msg Msg) (image.Point, bool) {
	switch message := msg.(type) {
	case TickMsg:
		return message.Size, true
	case window.Resize:
		return message.Size, true
	default:
		return image.Point{}, false
	}
}
