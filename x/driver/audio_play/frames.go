package audio_play

import pcs "github.com/lewtec/lewkit/x/sound"

// Frames is an io.WriteCloser that forwards whole PCM frames to write.
// A trailing partial frame makes Close return [pcs.ErrFrame] after end.
type Frames struct {
	frame  int
	tail   []byte
	closed bool
	write  func([]byte) error
	end    func() error
}

// NewFrames returns a writer for interleaved frames of frame bytes.
func NewFrames(frame int, write func([]byte) error, end func() error) *Frames {
	return &Frames{frame: frame, write: write, end: end}
}

// Write buffers a short tail and forwards every complete frame.
func (w *Frames) Write(p []byte) (int, error) {
	if w.closed {
		return 0, pcs.ErrClosed
	}
	buf := append(append([]byte(nil), w.tail...), p...)
	n := len(buf) - len(buf)%w.frame
	if n > 0 {
		if err := w.write(buf[:n]); err != nil {
			return 0, err
		}
	}
	w.tail = append(w.tail[:0], buf[n:]...)
	return len(p), nil
}

// Close finishes the device, then reports a queued partial frame.
func (w *Frames) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	if err := w.end(); err != nil {
		return err
	}
	if len(w.tail) != 0 {
		return pcs.ErrFrame
	}
	return nil
}
