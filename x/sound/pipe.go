package sound

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"time"
)

// ErrSeek means a frame position is outside the stream.
var ErrSeek = errors.New("sound seek")

// Pipeline is a pull chain of interleaved PCM. Seek moves by whole frames.
// Gain and Take keep that position. Merge seeks every input together.
type Pipeline struct {
	stage stage
}

type stage interface {
	Read([]byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Format() Format
	Frames() int64
}

// FromSeeker reads interleaved PCM from r. frames is the length.
// A negative frames value means the length is unknown and SeekEnd fails.
func FromSeeker(format Format, frames int64, r io.ReadSeeker) (*Pipeline, error) {
	frame, err := format.Frame()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrFormat
	}
	length := frames
	if frames < 0 {
		length = -1
	}
	stage := &pcmSeek{format: format, r: r, frame: frame, length: length}
	if _, err := stage.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return &Pipeline{stage: stage}, nil
}

// New returns a pipeline over pcm. The length must be a whole number of frames.
func New(format Format, pcm []byte) (*Pipeline, error) {
	frame, err := format.Frame()
	if err != nil {
		return nil, err
	}
	if len(pcm)%frame != 0 {
		return nil, ErrFrame
	}
	buf := append([]byte(nil), pcm...)
	return &Pipeline{stage: &clip{format: format, data: buf, frame: frame, length: int64(len(buf) / frame)}}, nil
}

// Format is the PCM layout of Read.
func (p *Pipeline) Format() Format { return p.stage.Format() }

// Frames is the number of frames Seek can address, including the end.
func (p *Pipeline) Frames() int64 { return p.stage.Frames() }

// Duration is Frames at the stream rate.
func (p *Pipeline) Duration() time.Duration {
	return p.Format().Duration(p.Frames())
}

// Read returns whole frames. A buffer shorter than one frame returns
// [io.ErrShortBuffer].
func (p *Pipeline) Read(b []byte) (int, error) { return p.stage.Read(b) }

// Seek sets the next frame. whence is [io.SeekStart], [io.SeekCurrent], or
// [io.SeekEnd]. The offset is a frame count.
func (p *Pipeline) Seek(offset int64, whence int) (int64, error) {
	return p.stage.Seek(offset, whence)
}

// Gain multiplies later samples by amp and clips to the format range.
func (p *Pipeline) Gain(amp float64) *Pipeline {
	p.stage = &gain{src: p.stage, amp: float32(amp)}
	return p
}

// Take keeps length frames starting at start. The new position is the start
// of that window.
func (p *Pipeline) Take(start, length int64) error {
	if start < 0 || length < 0 || start+length > p.Frames() {
		return ErrSeek
	}
	if _, err := p.stage.Seek(start, io.SeekStart); err != nil {
		return err
	}
	p.stage = &window{src: p.stage, start: start, length: length}
	return nil
}

// Merge sums pipelines of the same format. A shorter input is silence after
// it ends. The result length is the longest input.
func Merge(in ...*Pipeline) (*Pipeline, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, ErrFormat
	}
	format := in[0].Format()
	frame, err := format.Frame()
	if err != nil {
		return nil, err
	}
	srcs := make([]stage, len(in))
	var length int64
	for i, pipe := range in {
		if pipe == nil || pipe.Format() != format {
			return nil, ErrFormat
		}
		srcs[i] = pipe.stage
		if n := pipe.Frames(); n > length {
			length = n
		}
		if _, err := pipe.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return &Pipeline{stage: &merged{format: format, srcs: srcs, frame: frame, length: length}}, nil
}

// FramesFor converts d into a frame index at f.Rate.
func (f Format) FramesFor(d time.Duration) (int64, error) {
	if err := f.Validate(); err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, ErrSeek
	}
	return int64(d) * int64(f.Rate) / int64(time.Second), nil
}

// Duration converts a frame count into a duration at f.Rate.
func (f Format) Duration(frames int64) time.Duration {
	if frames <= 0 || f.Rate <= 0 {
		return 0
	}
	return time.Duration(frames) * time.Second / time.Duration(f.Rate)
}

type cursor struct {
	length int64
	at     int64
}

func (c cursor) place(offset int64, whence int) (cursor, error) {
	next := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		next = c.at + offset
	case io.SeekEnd:
		next = c.length + offset
	default:
		return c, ErrSeek
	}
	if next < 0 || next > c.length {
		return c, ErrSeek
	}
	c.at = next
	return c, nil
}

type pcmSeek struct {
	format Format
	r      io.ReadSeeker
	frame  int
	length int64
	at     int64
}

func (s *pcmSeek) Format() Format { return s.format }
func (s *pcmSeek) Frames() int64  { return s.length }

func (s *pcmSeek) Read(p []byte) (int, error) {
	if s.length >= 0 && s.at >= s.length {
		return 0, io.EOF
	}
	if len(p) < s.frame {
		return 0, io.ErrShortBuffer
	}
	n := len(p) - len(p)%s.frame
	if s.length >= 0 {
		max := int(s.length-s.at) * s.frame
		if n > max {
			n = max
		}
	}
	got, err := io.ReadFull(s.r, p[:n])
	frames := got / s.frame
	s.at += int64(frames)
	if frames == 0 {
		if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
			return 0, io.EOF
		}
		return 0, err
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	return frames * s.frame, err
}

func (s *pcmSeek) Seek(offset int64, whence int) (int64, error) {
	if s.length < 0 {
		pos, err := s.r.Seek(offset*int64(s.frame), whence)
		if err != nil {
			return 0, err
		}
		if pos%int64(s.frame) != 0 {
			return 0, ErrFrame
		}
		s.at = pos / int64(s.frame)
		return s.at, nil
	}
	next, err := (cursor{length: s.length, at: s.at}).place(offset, whence)
	if err != nil {
		return 0, err
	}
	if _, err := s.r.Seek(next.at*int64(s.frame), io.SeekStart); err != nil {
		return 0, err
	}
	s.at = next.at
	return s.at, nil
}

type clip struct {
	format Format
	data   []byte
	frame  int
	cursor
}

func (c *clip) Format() Format { return c.format }
func (c *clip) Frames() int64  { return c.length }

func (c *clip) Read(p []byte) (int, error) {
	if c.at >= c.length {
		return 0, io.EOF
	}
	if len(p) < c.frame {
		return 0, io.ErrShortBuffer
	}
	n := int(c.length-c.at) * c.frame
	if n > len(p) {
		n = len(p) - len(p)%c.frame
	}
	off := int(c.at) * c.frame
	copy(p[:n], c.data[off:off+n])
	c.at += int64(n / c.frame)
	return n, nil
}

func (c *clip) Seek(offset int64, whence int) (int64, error) {
	next, err := c.cursor.place(offset, whence)
	if err != nil {
		return 0, err
	}
	c.cursor = next
	return c.at, nil
}

type gain struct {
	src stage
	amp float32
}

func (g *gain) Format() Format { return g.src.Format() }
func (g *gain) Frames() int64  { return g.src.Frames() }

func (g *gain) Read(p []byte) (int, error) {
	n, err := g.src.Read(p)
	if n > 0 {
		applyGain(g.Format(), p[:n], g.amp)
	}
	return n, err
}

func (g *gain) Seek(offset int64, whence int) (int64, error) {
	return g.src.Seek(offset, whence)
}

func applyGain(format Format, p []byte, amp float32) {
	if amp == 1 {
		return
	}
	switch format.Sample {
	case SampleS16LE:
		for off := 0; off+2 <= len(p); off += 2 {
			v := float32(int16(binary.LittleEndian.Uint16(p[off:]))) * amp
			v = min(v, math.MaxInt16)
			v = max(v, float32(math.MinInt16))
			binary.LittleEndian.PutUint16(p[off:], uint16(int16(v)))
		}
	case SampleF32LE:
		for off := 0; off+4 <= len(p); off += 4 {
			v := math.Float32frombits(binary.LittleEndian.Uint32(p[off:])) * amp
			v = min(v, 1)
			v = max(v, -1)
			binary.LittleEndian.PutUint32(p[off:], math.Float32bits(v))
		}
	}
}

type window struct {
	src    stage
	start  int64
	length int64
	at     int64
}

func (w *window) Format() Format { return w.src.Format() }
func (w *window) Frames() int64  { return w.length }

func (w *window) Read(p []byte) (int, error) {
	if w.at >= w.length {
		return 0, io.EOF
	}
	frame, err := w.Format().Frame()
	if err != nil {
		return 0, err
	}
	if len(p) < frame {
		return 0, io.ErrShortBuffer
	}
	limit := int(w.length-w.at) * frame
	if len(p) > limit {
		p = p[:limit]
	}
	n, err := w.src.Read(p)
	w.at += int64(n / frame)
	return n, err
}

func (w *window) Seek(offset int64, whence int) (int64, error) {
	next, err := (cursor{length: w.length, at: w.at}).place(offset, whence)
	if err != nil {
		return 0, err
	}
	if _, err := w.src.Seek(w.start+next.at, io.SeekStart); err != nil {
		return 0, err
	}
	w.at = next.at
	return w.at, nil
}

type merged struct {
	format Format
	srcs   []stage
	frame  int
	length int64
	at     int64
}

func (m *merged) Format() Format { return m.format }
func (m *merged) Frames() int64  { return m.length }

func (m *merged) Seek(offset int64, whence int) (int64, error) {
	next, err := (cursor{length: m.length, at: m.at}).place(offset, whence)
	if err != nil {
		return 0, err
	}
	for _, src := range m.srcs {
		target := next.at
		if target > src.Frames() {
			target = src.Frames()
		}
		if _, err := src.Seek(target, io.SeekStart); err != nil {
			return 0, err
		}
	}
	m.at = next.at
	return m.at, nil
}

func (m *merged) Read(p []byte) (int, error) {
	if m.at >= m.length {
		return 0, io.EOF
	}
	if len(p) < m.frame {
		return 0, io.ErrShortBuffer
	}
	nframes := len(p) / m.frame
	if remain := int(m.length - m.at); nframes > remain {
		nframes = remain
	}
	bufs := make([][]byte, len(m.srcs))
	produced := 0
	for i, src := range m.srcs {
		buf := make([]byte, nframes*m.frame)
		n, err := io.ReadFull(src, buf)
		frames := n / m.frame
		bufs[i] = buf[:frames*m.frame]
		if frames > produced {
			produced = frames
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return 0, err
		}
	}
	if produced == 0 {
		return 0, io.EOF
	}
	for _, src := range m.srcs {
		target := m.at + int64(produced)
		if target > src.Frames() {
			target = src.Frames()
		}
		if _, err := src.Seek(target, io.SeekStart); err != nil {
			return 0, err
		}
	}
	for frame := 0; frame < produced; frame++ {
		heads := make([][]byte, 0, len(bufs))
		off := frame * m.frame
		for _, buf := range bufs {
			if off+m.frame <= len(buf) {
				heads = append(heads, buf[off:off+m.frame])
			}
		}
		mixFrame(m.format, p[off:off+m.frame], heads)
	}
	m.at += int64(produced)
	return produced * m.frame, nil
}
