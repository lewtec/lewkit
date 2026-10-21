package avformat

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openHEVC(data []byte) (Demuxer, error) {
	parts := splitHEVCAccessUnits(data)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: hevc", avutil.ErrInvalidData)
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
				Type: avutil.MediaVideo, ID: avcodec.IDHEVC,
				Format: avutil.PixYUV420P, TimeBase: avutil.Q(1, 25),
			},
			TimeBase: avutil.Q(1, 25),
		}},
		pkts: pkts,
	}, nil
}

// splitHEVCAccessUnits cuts annex B on each VPS, which is how this muxer starts a frame.
func splitHEVCAccessUnits(data []byte) [][]byte {
	var starts []int
	for i := 0; i+5 < len(data); i++ {
		if data[i] != 0 || data[i+1] != 0 {
			continue
		}
		off := 0
		if data[i+2] == 1 {
			off = 3
		} else if i+5 < len(data) && data[i+2] == 0 && data[i+3] == 1 {
			off = 4
		}
		if off == 0 || i+off >= len(data) {
			continue
		}
		if int(data[i+off]>>1)&0x3f == 32 {
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
