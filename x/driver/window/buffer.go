package window

import (
	"context"
	"image"
	"image/draw"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/event"
)

// Buffer is a pair of *image.RGBA pages. Frame is the back buffer.
// Draw swaps back and front; only the last swap is shown (UDP).
type Buffer struct {
	mu          sync.Mutex
	back, front *image.RGBA
	closed      bool
	bus         *event.Bus[Event]
	period      time.Duration
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

// FramePeriod is the display interval. Zero on the buffer means [DefaultFramePeriod].
func (b *Buffer) FramePeriod() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.period <= 0 {
		return DefaultFramePeriod
	}
	return b.period
}

// SetFramePeriod records the display interval. d <= 0 means [DefaultFramePeriod].
func (b *Buffer) SetFramePeriod(d time.Duration) {
	if d <= 0 {
		d = DefaultFramePeriod
	}
	b.mu.Lock()
	b.period = d
	b.mu.Unlock()
}

// Size is the back buffer size. Hosts that track a window size override this.
func (b *Buffer) Size() image.Point {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.back == nil {
		return image.Point{}
	}
	return b.back.Rect.Size()
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

// SwapBlit swaps, then calls blit when the front page is live.
func SwapBlit(b *Buffer, blit func() error) error {
	if err := b.Swap(); err != nil {
		return err
	}
	if b.Front() == nil {
		return nil
	}
	return blit()
}

// WithFront runs fn with the front page. Swap waits until fn returns.
func (b *Buffer) WithFront(fn func(*image.RGBA)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fn(b.front)
}

// Resize replaces both pages and emits Resize. Overlapping pixels are copied.
func (b *Buffer) Resize(size image.Point) error {
	changed, err := b.EnsureSize(size)
	if err != nil {
		return err
	}
	if changed {
		b.bus.Publish(Resize{Size: size})
	}
	return nil
}

// EnsureSize replaces both pages if needed. It does not emit Resize.
func (b *Buffer) EnsureSize(size image.Point) (bool, error) {
	if size.X <= 0 || size.Y <= 0 {
		return false, ErrSize
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return false, ErrClosed
	}
	if b.back.Rect.Size() == size {
		return false, nil
	}
	b.back = resizeRGBA(b.back, size.X, size.Y)
	b.front = resizeRGBA(b.front, size.X, size.Y)
	return true, nil
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
