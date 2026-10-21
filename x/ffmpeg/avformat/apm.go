package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openAPM(b []byte) (Demuxer, error) {
	if len(b) < 100 || binary.LittleEndian.Uint16(b[0:2]) != 0x2000 {
		return nil, fmt.Errorf("%w: apm", avutil.ErrInvalidData)
	}
	channels := int(binary.LittleEndian.Uint16(b[2:4]))
	if channels != 1 {
		return nil, fmt.Errorf("%w: apm channels", avutil.ErrInvalidData)
	}
	rate := int(binary.LittleEndian.Uint32(b[4:8]))
	if rate <= 0 {
		return nil, fmt.Errorf("%w: apm rate", avutil.ErrInvalidData)
	}
	if string(b[20:24]) != "vs12" || string(b[96:100]) != "DATA" {
		return nil, fmt.Errorf("%w: apm tag", avutil.ErrInvalidData)
	}
	dataSize := int(binary.LittleEndian.Uint32(b[28:32]))
	data := b[100:]
	if dataSize >= 0 && dataSize < len(data) {
		data = data[:dataSize]
	}
	tb := avutil.Q(1, int32(rate))
	pkts := []*avutil.Packet{{
		Data: append([]byte(nil), data...), Duration: int64(len(data) * 2),
		TimeBase: tb, Key: true,
	}}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDADPCMIMAAPM, SampleFormat: avutil.SampleS16,
			SampleRate: rate, Layout: avutil.Mono(), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

// writeAPM builds a seekable Rayman 2 APM file. The 20-byte waveform
// header is followed by an 80-byte vs12 block and the raw ADPCM bytes.
func writeAPM(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMIMAAPM {
		return nil, fmt.Errorf("%w: apm", avutil.ErrUnsupported)
	}
	if streams[0].Par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: apm channels", avutil.ErrInvalidData)
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
	body := make([]byte, 100+len(data))
	binary.LittleEndian.PutUint16(body[0:], 0x2000)
	binary.LittleEndian.PutUint16(body[2:], 1)
	binary.LittleEndian.PutUint32(body[4:], uint32(rate))
	binary.LittleEndian.PutUint32(body[8:], uint32(rate*2))
	binary.LittleEndian.PutUint16(body[12:], 1024)
	binary.LittleEndian.PutUint16(body[14:], 4)
	binary.LittleEndian.PutUint32(body[16:], 80)
	copy(body[20:], "vs12")
	binary.LittleEndian.PutUint32(body[24:], uint32(len(body)))
	binary.LittleEndian.PutUint32(body[28:], uint32(len(data)))
	binary.LittleEndian.PutUint32(body[32:], 0xFFFFFFFF)
	copy(body[96:], "DATA")
	copy(body[100:], data)
	return body, nil
}
