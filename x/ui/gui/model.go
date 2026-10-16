package gui

import (
	"context"
	"image"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
)

// Msg is an incoming event. Host messages are [window.Resize],
// [window.Expose], [window.Close], [window.Pointer], [window.Scroll],
// [window.Key], [window.Drop], [window.Density], [window.FontScale], [TickMsg], [Cover], and [ModeMsg].
type Msg any

// Cmd produces a follow-up [Msg]. Nil means no command.
// Run calls it with the context that started the window.
type Cmd func(context.Context) Msg

// Tick immediately produces a [TickMsg]. Use from Init for the first frame.
func Tick() Cmd {
	return func(context.Context) Msg { return TickMsg{} }
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
	return func(ctx context.Context) Msg {
		if wait := time.Until(deadline); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
			}
		}
		return TickMsg{}
	}
}

// quitMsg ends the window loop without canceling the caller's context.
type quitMsg struct{}

// Model is the bubbletea trio. View is a layout [Node], not a tensor.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() Node
}

// TickMsg is the clock. Models request it with [Tick] and [Every]; Run
// does not tick on its own. Pointer, Scroll, Resize, and Key stay the
// messages [window] emits. Size is the window client size in device
// pixels. Scale is device pixels per logical pixel. Font is the user's
// text-size preference. Zero means one for either factor.
// Dead is the client area a bar or notch covers. Nil means none.
// Period is [window.Window.FramePeriod]. FPS is the smoothed
// frames-per-second since the previous tick.
type TickMsg struct {
	Elapsed time.Duration
	Size    image.Point
	Scale   float32
	Font    float32
	Dead    []image.Rectangle
	FPS     float64
	Period  time.Duration
}

// Cover is the client area a bar or notch covers, in device pixels.
// Run delivers it when that set changes. Nil zones means none.
type Cover struct {
	Zones []image.Rectangle
}

// Metrics is the scale, text preference, and covered zones of the frame
// on screen. Observe copies them from a tick, a density change, or [Cover].
// Hit lays the view out again with those values.
type Metrics struct {
	Scale float32
	Font  float32
	Dead  []image.Rectangle
}

// Observe stores the frame metrics carried by msg.
func (m *Metrics) Observe(msg Msg) {
	if m == nil {
		return
	}
	switch event := msg.(type) {
	case TickMsg:
		m.Scale = event.Scale
		m.Font = event.Font
		m.Dead = event.Dead
	case Cover:
		m.Dead = event.Zones
	case window.Density:
		m.Scale = float32(event.Scale.Factor())
	case window.FontScale:
		m.Font = float32(event.Scale.Factor())
	}
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
