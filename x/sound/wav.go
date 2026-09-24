package sound

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	wavPCM   = 1
	wavFloat = 3
	wavMax   = 256 << 20
)

// WriteWAV writes a RIFF WAVE of interleaved PCM.
func WriteWAV(w io.Writer, format Format, pcm []byte) error {
	frame, err := format.Frame()
	if err != nil {
		return err
	}
	if len(pcm)%frame != 0 {
		return ErrFrame
	}
	tag := uint16(wavPCM)
	if format.Sample == SampleF32LE {
		tag = wavFloat
	}
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+len(pcm)))
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], tag)
	binary.LittleEndian.PutUint16(header[22:], uint16(format.Channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(format.Rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(format.Rate*frame))
	binary.LittleEndian.PutUint16(header[32:], uint16(frame))
	binary.LittleEndian.PutUint16(header[34:], uint16(format.Width()*8))
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(pcm)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err = w.Write(pcm)
	return err
}

// ReadWAV reads one PCM RIFF WAVE.
func ReadWAV(r io.Reader) (Format, []byte, error) {
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Format{}, nil, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return Format{}, nil, ErrFormat
	}
	var format Format
	var sawFmt bool
	var pcm []byte
	for {
		var chunk [8]byte
		if _, err := io.ReadFull(r, chunk[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return Format{}, nil, err
		}
		size := binary.LittleEndian.Uint32(chunk[4:])
		if size > wavMax {
			return Format{}, nil, ErrFormat
		}
		body := make([]byte, size)
		if _, err := io.ReadFull(r, body); err != nil {
			return Format{}, nil, err
		}
		switch string(chunk[0:4]) {
		case "fmt ":
			parsed, err := parseFmt(body)
			if err != nil {
				return Format{}, nil, err
			}
			format = parsed
			sawFmt = true
		case "data":
			pcm = body
		}
		if size%2 == 1 {
			if _, err := io.ReadFull(r, chunk[:1]); err != nil {
				return Format{}, nil, err
			}
		}
	}
	if !sawFmt || pcm == nil {
		return Format{}, nil, ErrFormat
	}
	frame, err := format.Frame()
	if err != nil {
		return Format{}, nil, err
	}
	if len(pcm)%frame != 0 {
		return Format{}, nil, ErrFrame
	}
	return format, pcm, nil
}

func parseFmt(body []byte) (Format, error) {
	if len(body) < 16 {
		return Format{}, ErrFormat
	}
	tag := binary.LittleEndian.Uint16(body[0:])
	format := Format{
		Channels: int(binary.LittleEndian.Uint16(body[2:])),
		Rate:     int(binary.LittleEndian.Uint32(body[4:])),
	}
	bits := binary.LittleEndian.Uint16(body[14:])
	switch {
	case tag == wavPCM && bits == 16:
		format.Sample = SampleS16LE
	case tag == wavFloat && bits == 32:
		format.Sample = SampleF32LE
	default:
		return Format{}, fmt.Errorf("%w: wav encoding", ErrFormat)
	}
	return format, nil
}
