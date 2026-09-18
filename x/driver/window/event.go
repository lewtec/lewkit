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
