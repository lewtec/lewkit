package window

import (
	"image"
	"image/draw"
	"sync"
)

// Buffer is an *image.RGBA that Resize reallocates.
type Buffer struct {
	mu     sync.Mutex
	img    *image.RGBA
	closed bool
}

// NewBuffer returns a w×h buffer.
func NewBuffer(w, h int) *Buffer {
	return &Buffer{img: image.NewRGBA(image.Rect(0, 0, w, h))}
}

// Frame is the current backing store.
func (b *Buffer) Frame() *image.RGBA {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.img
}

// Resize replaces the buffer. Overlapping pixels are copied.
func (b *Buffer) Resize(size image.Point) error {
	if size.X <= 0 || size.Y <= 0 {
		return ErrSize
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrClosed
	}
	b.img = resizeRGBA(b.img, size.X, size.Y)
	return nil
}

// Close marks the buffer closed. Further Resize returns [ErrClosed].
func (b *Buffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
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
