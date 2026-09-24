package sound

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/sniff"
)

var (
	// ErrNil is Register with a nil decoder.
	ErrNil = errors.New("nil decoder")
	// ErrExist is Register when the name is already in the registry.
	ErrExist = errors.New("decoder already registered")
)

// Decoder identifies an audio file and turns it into a pipeline.
type Decoder interface {
	Name() string
	Extensions() []string
	Magic() [][]byte
	Decode(r io.Reader) (*Pipeline, error)
}

var std = &Registry{}

func init() {
	MustRegister(wavDecoder{})
}

// Register adds a decoder to the process-wide registry.
func Register(d Decoder) error {
	if d == nil {
		return ErrNil
	}
	if sniff.HasName(std.list, d.Name()) {
		return fmt.Errorf("%s: %w", d.Name(), ErrExist)
	}
	std.list = append(std.list, d)
	return nil
}

// MustRegister is Register that panics on error.
func MustRegister(d Decoder) {
	if err := Register(d); err != nil {
		panic(err)
	}
}

// Detect finds a decoder. A matching extension wins over magic.
func Detect(name string, magic []byte) (Decoder, bool) {
	return std.Detect(name, magic)
}

// Decode sniffs name and the start of r, then decodes the whole stream.
// A seekable r stays seekable for the decoder.
func Decode(name string, r io.Reader) (*Pipeline, error) {
	magic, r, err := readHead(r)
	if err != nil {
		return nil, err
	}
	dec, ok := Detect(name, magic)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrFormat, name)
	}
	return dec.Decode(r)
}

// Registry looks up decoders by extension or magic.
type Registry struct {
	list []Decoder
}

// NewRegistry builds a registry. The slice is copied.
func NewRegistry(list ...Decoder) *Registry {
	return &Registry{list: append([]Decoder(nil), list...)}
}

// Detect returns a decoder for name or magic.
func (r *Registry) Detect(name string, magic []byte) (Decoder, bool) {
	if d, ok := r.byExtension(name); ok {
		return d, true
	}
	return r.byMagic(magic)
}

func (r *Registry) byExtension(name string) (Decoder, bool) {
	return sniff.ByExtension(r.list, name)
}

func (r *Registry) byMagic(p []byte) (Decoder, bool) {
	return sniff.ByMagic(r.list, p)
}

func readHead(r io.Reader) ([]byte, io.Reader, error) {
	var buf [64]byte
	if rs, ok := r.(io.ReadSeeker); ok {
		n, err := io.ReadFull(rs, buf[:])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, nil, err
		}
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, nil, err
		}
		return buf[:n], rs, nil
	}
	n, err := io.ReadFull(r, buf[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, nil, err
	}
	return buf[:n], io.MultiReader(bytes.NewReader(buf[:n]), r), nil
}

type wavDecoder struct{}

func (wavDecoder) Name() string { return "wav" }

func (wavDecoder) Extensions() []string { return []string{".wav"} }

func (wavDecoder) Magic() [][]byte { return [][]byte{[]byte("RIFF")} }

func (wavDecoder) Decode(r io.Reader) (*Pipeline, error) { return decodeWAV(r) }

func decodeWAV(r io.Reader) (*Pipeline, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		format, pcm, err := ReadWAV(r)
		if err != nil {
			return nil, err
		}
		return New(format, pcm)
	}
	format, dataAt, size, err := findWAVData(rs)
	if err != nil {
		return nil, err
	}
	frame, err := format.Frame()
	if err != nil {
		return nil, err
	}
	if size%int64(frame) != 0 {
		return nil, ErrFrame
	}
	return FromSeeker(format, size/int64(frame), &spanReader{r: rs, start: dataAt, size: size})
}

func findWAVData(r io.ReadSeeker) (Format, int64, int64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return Format{}, 0, 0, err
	}
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Format{}, 0, 0, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return Format{}, 0, 0, ErrFormat
	}
	var format Format
	var sawFmt bool
	for {
		var chunk [8]byte
		if _, err := io.ReadFull(r, chunk[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return Format{}, 0, 0, err
		}
		size := int64(binary.LittleEndian.Uint32(chunk[4:]))
		if size > wavMax {
			return Format{}, 0, 0, ErrFormat
		}
		switch string(chunk[0:4]) {
		case "fmt ":
			body := make([]byte, size)
			if _, err := io.ReadFull(r, body); err != nil {
				return Format{}, 0, 0, err
			}
			parsed, err := parseFmt(body)
			if err != nil {
				return Format{}, 0, 0, err
			}
			format = parsed
			sawFmt = true
		case "data":
			if !sawFmt {
				return Format{}, 0, 0, ErrFormat
			}
			at, err := r.Seek(0, io.SeekCurrent)
			if err != nil {
				return Format{}, 0, 0, err
			}
			return format, at, size, nil
		default:
			if _, err := r.Seek(size, io.SeekCurrent); err != nil {
				return Format{}, 0, 0, err
			}
		}
		if size%2 == 1 {
			if _, err := r.Seek(1, io.SeekCurrent); err != nil {
				return Format{}, 0, 0, err
			}
		}
	}
	return Format{}, 0, 0, ErrFormat
}

type spanReader struct {
	r     io.ReadSeeker
	start int64
	size  int64
	at    int64
}

func (s *spanReader) Read(p []byte) (int, error) {
	if s.at >= s.size {
		return 0, io.EOF
	}
	if int64(len(p)) > s.size-s.at {
		p = p[:s.size-s.at]
	}
	n, err := s.r.Read(p)
	s.at += int64(n)
	return n, err
}

func (s *spanReader) Seek(offset int64, whence int) (int64, error) {
	next, err := (cursor{length: s.size, at: s.at}).place(offset, whence)
	if err != nil {
		return 0, err
	}
	if _, err := s.r.Seek(s.start+next.at, io.SeekStart); err != nil {
		return 0, err
	}
	s.at = next.at
	return s.at, nil
}
