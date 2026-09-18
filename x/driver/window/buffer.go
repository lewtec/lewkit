package window

import (
	"context"
	"image"
	"image/draw"
	"sync"

	"github.com/lewtec/lewkit/x/event"
)

// Buffer is a pair of *image.RGBA pages. Frame is the back buffer.
// Draw swaps back and front; only the last swap is shown (UDP).
type Buffer struct {
	mu          sync.Mutex
	back, front *image.RGBA
	closed      bool
	bus         *event.Bus[Event]
}

// NewBuffer returns a w×h double buffer.
func NewBuffer(w, h int) *Buffer {
	return &Buffer{
		back:  image.NewRGBA(image.Rect(0, 0, w, h)),
		front: image.NewRGBA(image.Rect(0, 0, w, h)),
		bus:   event.New[Event](),
	}
}

// Subscribe is [event.Bus.Subscribe] for this window.
func (b *Buffer) Subscribe(ctx context.Context) <-chan Event {
	return b.bus.Subscribe(ctx)
}

// Emit sends ev to subscribers. Hosts call it for Expose.
func (b *Buffer) Emit(ev Event) {
	b.bus.Publish(ev)
}

// Frame is the back buffer. After Draw, the next Frame is the other page.
func (b *Buffer) Frame() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.back
}

// Swap publishes the back buffer as front. The previous front becomes back.
func (b *Buffer) Swap() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrClosed
	}
	b.back, b.front = b.front, b.back
	return nil
}

// Front is the last swapped page. Hosts blit this on their next turn.
func (b *Buffer) Front() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.front
}

// WithFront runs fn with the front page. Swap waits until fn returns.
func (b *Buffer) WithFront(fn func(*image.RGBA)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fn(b.front)
}

// Resize replaces both pages. Overlapping pixels are copied.
func (b *Buffer) Resize(size image.Point) error {
	if size.X <= 0 || size.Y <= 0 {
		return ErrSize
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	old := b.back.Rect.Size()
	if old == size {
		b.mu.Unlock()
		return nil
	}
	b.back = resizeRGBA(b.back, size.X, size.Y)
	b.front = resizeRGBA(b.front, size.X, size.Y)
	b.mu.Unlock()
	b.bus.Publish(Resize{Size: size})
	return nil
}

// Close marks the buffer closed. Further Resize returns [ErrClosed].
func (b *Buffer) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	b.bus.Publish(Close{})
	return nil
}

// Closed reports whether Close has been called.
func (b *Buffer) Closed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

func resizeRGBA(src *image.RGBA, w, h int) *image.RGBA {
	if src != nil && src.Rect.Dx() == w && src.Rect.Dy() == h {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if src != nil {
		draw.Draw(dst, dst.Bounds(), src, src.Rect.Min, draw.Src)
	}
	return dst
}

// ToBGRA writes src as packed BGRA into dst. dst must be at least len(src.Pix).
func ToBGRA(dst []byte, src *image.RGBA) {
	n := copy(dst, src.Pix)
	for i := 0; i+3 < n; i += 4 {
		dst[i], dst[i+2] = dst[i+2], dst[i]
	}
}
