package sound

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
)

// Mix sums srcs into dst. Each source is interleaved PCM in format.
// A source that ends contributes silence. A trailing partial frame is dropped.
func Mix(format Format, dst io.Writer, srcs ...io.Reader) error {
	frame, err := format.Frame()
	if err != nil {
		return err
	}
	if dst == nil {
		return ErrFormat
	}
	pending := make([][]byte, len(srcs))
	tmp := make([]byte, frame)
	one := make([]byte, frame)
	heads := make([][]byte, 0, len(srcs))
	for {
		heads = heads[:0]
		for i, src := range srcs {
			for src != nil && len(pending[i]) < frame {
				n, err := src.Read(tmp)
				if n > 0 {
					pending[i] = append(pending[i], tmp[:n]...)
				}
				if err == io.EOF {
					src = nil
					srcs[i] = nil
					break
				}
				if err != nil {
					return err
				}
				if n == 0 {
					return io.ErrNoProgress
				}
			}
			if len(pending[i]) >= frame {
				heads = append(heads, pending[i][:frame])
			}
		}
		if len(heads) == 0 {
			return nil
		}
		mixFrame(format, one, heads)
		if _, err := dst.Write(one); err != nil {
			return err
		}
		for i := range pending {
			if len(pending[i]) >= frame {
				pending[i] = append([]byte(nil), pending[i][frame:]...)
			}
		}
	}
}

// Mixer sums several PCM writers into one reader.
type Mixer struct {
	format Format
	frame  int
	mu     sync.Mutex
	cond   *sync.Cond
	tracks []*track
	heads  [][]byte
	closed bool
}

// NewMixer validates format and returns a mixer.
func NewMixer(format Format) (*Mixer, error) {
	frame, err := format.Frame()
	if err != nil {
		return nil, err
	}
	m := &Mixer{format: format, frame: frame}
	m.cond = sync.NewCond(&m.mu)
	return m, nil
}

// Add returns a writer mixed into later Read calls.
func (m *Mixer) Add() (io.WriteCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	t := &track{m: m}
	m.tracks = append(m.tracks, t)
	m.cond.Broadcast()
	return t, nil
}

// Close drops every track after its queued whole frames are still readable.
func (m *Mixer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	for _, t := range m.tracks {
		t.closed = true
		if extra := len(t.buf) % m.frame; extra != 0 {
			t.buf = t.buf[:len(t.buf)-extra]
		}
	}
	m.cond.Broadcast()
	return nil
}

// Read fills p with mixed frames. It blocks until one track has a frame
// or no further frame can arrive. p shorter than one frame returns
// [io.ErrShortBuffer].
func (m *Mixer) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(p) < m.frame {
		return 0, io.ErrShortBuffer
	}
	n := 0
	for n+m.frame <= len(p) {
		for !m.ready() {
			if m.finished() {
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}
			m.cond.Wait()
		}
		m.mixLocked(p[n : n+m.frame])
		n += m.frame
	}
	return n, nil
}

func (m *Mixer) ready() bool {
	for _, t := range m.tracks {
		if len(t.buf) >= m.frame {
			return true
		}
	}
	return false
}

func (m *Mixer) finished() bool {
	if m.ready() {
		return false
	}
	if m.closed {
		return true
	}
	if len(m.tracks) == 0 {
		return false
	}
	for _, t := range m.tracks {
		if !t.closed {
			return false
		}
	}
	return true
}

func (m *Mixer) mixLocked(dst []byte) {
	m.heads = m.heads[:0]
	for _, t := range m.tracks {
		if len(t.buf) >= m.frame {
			m.heads = append(m.heads, t.buf[:m.frame])
		}
	}
	mixFrame(m.format, dst, m.heads)
	for _, t := range m.tracks {
		if len(t.buf) >= m.frame {
			t.buf = t.buf[m.frame:]
		}
	}
	out := m.tracks[:0]
	for _, t := range m.tracks {
		if t.closed && len(t.buf) < m.frame {
			t.buf = nil
			continue
		}
		out = append(out, t)
	}
	m.tracks = out
}

type track struct {
	m      *Mixer
	buf    []byte
	closed bool
}

func (t *track) Write(p []byte) (int, error) {
	t.m.mu.Lock()
	defer t.m.mu.Unlock()
	if t.closed || t.m.closed {
		return 0, ErrClosed
	}
	t.buf = append(t.buf, p...)
	t.m.cond.Broadcast()
	return len(p), nil
}

func (t *track) Close() error {
	t.m.mu.Lock()
	defer t.m.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	if extra := len(t.buf) % t.m.frame; extra != 0 {
		t.buf = t.buf[:len(t.buf)-extra]
	}
	t.m.cond.Broadcast()
	return nil
}

func mixFrame(format Format, dst []byte, srcs [][]byte) {
	switch format.Sample {
	case SampleS16LE:
		for off := 0; off < len(dst); off += 2 {
			var sum int32
			for _, src := range srcs {
				sum += int32(int16(binary.LittleEndian.Uint16(src[off:])))
			}
			sum = min(sum, math.MaxInt16)
			sum = max(sum, math.MinInt16)
			binary.LittleEndian.PutUint16(dst[off:], uint16(int16(sum)))
		}
	case SampleF32LE:
		for off := 0; off < len(dst); off += 4 {
			var sum float32
			for _, src := range srcs {
				sum += math.Float32frombits(binary.LittleEndian.Uint32(src[off:]))
			}
			sum = min(sum, 1)
			sum = max(sum, -1)
			binary.LittleEndian.PutUint32(dst[off:], math.Float32bits(sum))
		}
	}
}
