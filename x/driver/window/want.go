package window

import (
	"image"
	"sync"
)

// WantSize is the last host-reported client size.
type WantSize struct {
	Width, Height int
}

// Point is the recorded size when both sides are positive, otherwise fallback.
func (s WantSize) Point(fallback image.Point) image.Point {
	if s.Width > 0 && s.Height > 0 {
		return image.Pt(s.Width, s.Height)
	}
	return fallback
}

// Set records width×height. False if the size is invalid or unchanged.
func (s *WantSize) Set(width, height int) bool {
	if width < 1 || height < 1 {
		return false
	}
	changed := s.Width != width || s.Height != height
	s.Width, s.Height = width, height
	return changed
}

// SetWant records a host client size and emits Resize when it changes.
func SetWant(b *Buffer, mu *sync.Mutex, want *WantSize, width, height int) {
	mu.Lock()
	changed := want.Set(width, height)
	mu.Unlock()
	if changed {
		b.Emit(Resize{Size: image.Pt(width, height)})
	}
}

// HostSize locks mu and returns the recorded client size, or the buffer size.
func HostSize(b *Buffer, mu *sync.Mutex, want *WantSize) image.Point {
	mu.Lock()
	defer mu.Unlock()
	return want.Point(b.Size())
}

// HostFrame returns the back buffer, grown to size when both axes are positive.
func HostFrame(b *Buffer, size image.Point) *image.RGBA {
	if size.X > 0 && size.Y > 0 {
		_, err := b.EnsureSize(size)
		if err != nil {
			return b.Frame()
		}
	}
	return b.Frame()
}
