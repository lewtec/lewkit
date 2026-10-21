package avformat

import (
	"encoding/binary"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openBlob(id avcodec.ID, data []byte) (Demuxer, error) {
	par := avcodec.Parameters{Type: avutil.MediaAudio, ID: id, SampleFormat: avutil.SampleS16, Layout: avutil.Stereo(), SampleRate: 44100}
	if id == avcodec.IDVorbis {
		par.SampleFormat = avutil.SampleFLT
	}
	if id == avcodec.IDFLAC && len(data) >= 26 && string(data[:4]) == "fLaC" {
		bits := binary.BigEndian.Uint64(data[18:26])
		rate := int((bits >> 44) & 0xfffff)
		channels := int((bits>>41)&7) + 1
		if rate > 0 {
			par.SampleRate = rate
		}
		par.Layout = layoutOr(avutil.ChannelLayout{}, channels)
		if channels == 1 {
			par.Layout = avutil.Mono()
		}
	}
	tb := avutil.Q(1, int32(par.SampleRate))
	par.TimeBase = tb
	return &memDemux{streams: []Stream{{Par: par, TimeBase: tb}}, pkts: []*avutil.Packet{{
		Data: append([]byte(nil), data...), TimeBase: tb, Key: true, Duration: 0,
	}}}, nil
}

func writeBlob(pkts []*avutil.Packet) []byte {
	var out []byte
	for _, pkt := range pkts {
		out = append(out, pkt.Data...)
	}
	return out
}
