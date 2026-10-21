package avformat

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openGSM(data []byte) (Demuxer, error) {
	if len(data)%33 != 0 {
		data = data[:len(data)-len(data)%33]
	}
	var pkts []*avutil.Packet
	for off, i := 0, 0; off+33 <= len(data); off, i = off+33, i+1 {
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), data[off:off+33]...),
			PTS:  int64(i * 160), DTS: int64(i * 160), Duration: 160,
			TimeBase: avutil.Q(1, 8000),
		})
	}
	return &memDemux{
		streams: []Stream{{
			Par: avcodec.Parameters{
				Type: avutil.MediaAudio, ID: avcodec.IDGSM,
				SampleFormat: avutil.SampleS16, SampleRate: 8000,
				Layout: avutil.Mono(), TimeBase: avutil.Q(1, 8000),
			},
			TimeBase: avutil.Q(1, 8000),
		}},
		pkts: pkts,
	}, nil
}

func openG722(data []byte) (Demuxer, error) {
	tb := avutil.Q(1, 16000)
	return &memDemux{
		streams: []Stream{{
			Par: avcodec.Parameters{
				Type: avutil.MediaAudio, ID: avcodec.IDG722,
				SampleFormat: avutil.SampleS16, SampleRate: 16000,
				Layout: avutil.Mono(), TimeBase: tb,
			},
			TimeBase: tb,
		}},
		pkts: []*avutil.Packet{{
			Data: append([]byte(nil), data...), TimeBase: tb, Key: true,
			Duration: int64(len(data) * 2),
		}},
	}, nil
}

func openH264(data []byte) (Demuxer, error) {
	parts := splitAccessUnits(data)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: h264", avutil.ErrInvalidData)
	}
	var pkts []*avutil.Packet
	for i, part := range parts {
		pkts = append(pkts, &avutil.Packet{
			Data: part, Key: true, PTS: int64(i), DTS: int64(i), Duration: 1,
			TimeBase: avutil.Q(1, 25),
		})
	}
	return &memDemux{
		streams: []Stream{{
			Par: avcodec.Parameters{
				Type: avutil.MediaVideo, ID: avcodec.IDH264,
				Format: avutil.PixYUV420P, TimeBase: avutil.Q(1, 25),
			},
			TimeBase: avutil.Q(1, 25),
		}},
		pkts: pkts,
	}, nil
}

// splitAccessUnits cuts annex B on each SPS NAL, which is how this muxer starts a frame.
func splitAccessUnits(data []byte) [][]byte {
	var starts []int
	for i := 0; i+4 < len(data); i++ {
		if data[i] != 0 || data[i+1] != 0 {
			continue
		}
		off := 0
		if data[i+2] == 1 {
			off = 3
		} else if i+4 < len(data) && data[i+2] == 0 && data[i+3] == 1 {
			off = 4
		}
		if off == 0 || i+off >= len(data) {
			continue
		}
		if data[i+off]&0x1f == 7 {
			starts = append(starts, i)
			i += off
		}
	}
	if len(starts) == 0 && len(data) > 0 {
		return [][]byte{data}
	}
	var out [][]byte
	for i, s := range starts {
		e := len(data)
		if i+1 < len(starts) {
			e = starts[i+1]
		}
		out = append(out, data[s:e])
	}
	return out
}
