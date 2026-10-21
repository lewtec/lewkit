package avformat

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openCodec2(data []byte) (Demuxer, error) {
	if len(data) < 7 || data[0] != 0xC0 || data[1] != 0xDE || data[2] != 0xC2 {
		return nil, fmt.Errorf("%w: codec2 magic", avutil.ErrInvalidData)
	}
	if data[3] != 0 || data[4] < 8 || data[5] != 4 {
		return nil, fmt.Errorf("%w: codec2 mode", avutil.ErrUnsupported)
	}
	return openCodec2Raw(data[7:])
}

func openCodec2Raw(data []byte) (Demuxer, error) {
	const frame = 7
	data = data[:len(data)-len(data)%frame]
	tb := avutil.Q(1, 8000)
	var pkts []*avutil.Packet
	for off, i := 0, 0; off+frame <= len(data); off, i = off+frame, i+1 {
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), data[off:off+frame]...),
			PTS:  int64(i * 320), DTS: int64(i * 320), Duration: 320,
			TimeBase: tb,
		})
	}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDCODEC2, SampleFormat: avutil.SampleS16,
			SampleRate: 8000, Layout: avutil.Mono(), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

func writeCodec2(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDCODEC2 {
		return nil, fmt.Errorf("%w: codec2 streams", avutil.ErrInvalidData)
	}
	payload := writeBlob(pkts)
	out := make([]byte, 7+len(payload))
	copy(out, []byte{0xC0, 0xDE, 0xC2, 0x00, 0x08, 0x04, 0x00})
	copy(out[7:], payload)
	return out, nil
}
