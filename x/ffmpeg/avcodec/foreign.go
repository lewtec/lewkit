package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type mp3Codec struct{}

func (mp3Codec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	dec, err := gomp3.NewDecoder(bytes.NewReader(pkt.Data))
	if err != nil {
		return nil, fmt.Errorf("%w: mp3", avutil.ErrInvalidData)
	}
	raw, err := io.ReadAll(dec)
	if err != nil && len(raw) == 0 {
		return nil, fmt.Errorf("%w: mp3", avutil.ErrInvalidData)
	}
	if len(raw)%4 != 0 {
		raw = raw[:len(raw)-len(raw)%4]
	}
	samples := len(raw) / 4
	frame, err := avutil.NewAudio(avutil.SampleS16, dec.SampleRate(), avutil.Stereo(), samples)
	if err != nil {
		return nil, err
	}
	copy(frame.Data[0], raw)
	frame.Duration = int64(samples)
	frame.PTS = pkt.PTS
	return []*avutil.Frame{frame}, nil
}

func (mp3Codec) Encode(*avutil.Frame) ([]*avutil.Packet, error) {
	return nil, fmt.Errorf("%w: mp3 encoder", avutil.ErrUnsupported)
}
func (mp3Codec) Flush() ([]*avutil.Packet, error) { return nil, nil }

type vorbisCodec struct{}

func (vorbisCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	reader, err := oggvorbis.NewReader(bytes.NewReader(pkt.Data))
	if err != nil {
		return nil, fmt.Errorf("%w: vorbis", avutil.ErrInvalidData)
	}
	channels := reader.Channels()
	var pcm []float32
	buf := make([]float32, 4096)
	for {
		n, err := reader.Read(buf)
		pcm = append(pcm, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: vorbis", avutil.ErrInvalidData)
		}
	}
	if channels <= 0 {
		return nil, fmt.Errorf("%w: vorbis channels", avutil.ErrInvalidData)
	}
	samples := len(pcm) / channels
	frame, err := avutil.NewAudio(avutil.SampleFLT, reader.SampleRate(), layoutOf(avutil.ChannelLayout{}, channels), samples)
	if err != nil {
		return nil, err
	}
	for i, sample := range pcm {
		binary.LittleEndian.PutUint32(frame.Data[0][i*4:], math.Float32bits(sample))
	}
	frame.Duration = int64(samples)
	frame.PTS = pkt.PTS
	return []*avutil.Frame{frame}, nil
}

func (vorbisCodec) Encode(*avutil.Frame) ([]*avutil.Packet, error) {
	return nil, fmt.Errorf("%w: vorbis encoder", avutil.ErrUnsupported)
}
func (vorbisCodec) Flush() ([]*avutil.Packet, error) { return nil, nil }
