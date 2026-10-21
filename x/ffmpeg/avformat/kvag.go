package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openKVAG(b []byte) (Demuxer, error) {
	if len(b) < 14 || string(b[:4]) != "KVAG" {
		return nil, fmt.Errorf("%w: kvag", avutil.ErrInvalidData)
	}
	size := int(binary.LittleEndian.Uint32(b[4:8]))
	rate := int(binary.LittleEndian.Uint32(b[8:12]))
	if binary.LittleEndian.Uint16(b[12:14]) != 0 {
		return nil, fmt.Errorf("%w: kvag channels", avutil.ErrInvalidData)
	}
	if rate <= 0 {
		return nil, fmt.Errorf("%w: kvag rate", avutil.ErrInvalidData)
	}
	data := b[14:]
	if size >= 0 && size < len(data) {
		data = data[:size]
	}
	tb := avutil.Q(1, int32(rate))
	pkts := []*avutil.Packet{{
		Data: append([]byte(nil), data...), Duration: int64(len(data) * 2),
		TimeBase: tb, Key: true,
	}}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDADPCMIMASSI, SampleFormat: avutil.SampleS16,
			SampleRate: rate, Layout: avutil.Mono(), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

// writeKVAG builds a seekable Simon & Schuster VAG file.
// The 14-byte header is the KVAG tag, the payload size, the sample rate,
// and a stereo flag. Packets are concatenated raw.
func writeKVAG(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMIMASSI {
		return nil, fmt.Errorf("%w: kvag", avutil.ErrUnsupported)
	}
	if streams[0].Par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: kvag channels", avutil.ErrInvalidData)
	}
	rate := streams[0].Par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	var data []byte
	for _, pkt := range pkts {
		if pkt == nil || len(pkt.Data) == 0 {
			continue
		}
		data = append(data, pkt.Data...)
	}
	body := make([]byte, 14+len(data))
	copy(body[0:], "KVAG")
	binary.LittleEndian.PutUint32(body[4:], uint32(len(data)))
	binary.LittleEndian.PutUint32(body[8:], uint32(rate))
	copy(body[14:], data)
	return body, nil
}
