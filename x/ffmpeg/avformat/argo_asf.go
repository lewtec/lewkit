package avformat

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openArgoASF(b []byte) (Demuxer, error) {
	if len(b) < 44 || string(b[:4]) != "ASF\x00" {
		return nil, fmt.Errorf("%w: argo_asf", avutil.ErrInvalidData)
	}
	blocks := int(binary.LittleEndian.Uint32(b[24:28]))
	rate := int(binary.LittleEndian.Uint16(b[36:38]))
	if rate <= 0 {
		return nil, fmt.Errorf("%w: argo_asf rate", avutil.ErrInvalidData)
	}
	if binary.LittleEndian.Uint32(b[40:44])&2 != 0 {
		return nil, fmt.Errorf("%w: argo_asf channels", avutil.ErrInvalidData)
	}
	data := b[44:]
	if blocks >= 0 && blocks*17 < len(data) {
		data = data[:blocks*17]
	}
	tb := avutil.Q(1, int32(rate))
	pkts := []*avutil.Packet{{
		Data: append([]byte(nil), data...), Duration: int64(len(data) / 17 * 32),
		TimeBase: tb, Key: true,
	}}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDADPCMARGO, SampleFormat: avutil.SampleS16,
			SampleRate: rate, Layout: avutil.Mono(), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

// writeArgoASF builds a seekable Argonaut ASF file (version 2.1).
// The embedded name is the output basename, at most eight bytes.
func writeArgoASF(url string, streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMARGO {
		return nil, fmt.Errorf("%w: argo_asf", avutil.ErrUnsupported)
	}
	if streams[0].Par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: argo_asf channels", avutil.ErrInvalidData)
	}
	rate := streams[0].Par.SampleRate
	if rate <= 0 {
		rate = 22050
	}
	if rate > 65535 {
		return nil, fmt.Errorf("%w: argo_asf rate %d", avutil.ErrInvalidData, rate)
	}
	var data []byte
	for _, pkt := range pkts {
		if pkt == nil || len(pkt.Data) == 0 {
			continue
		}
		if len(pkt.Data)%17 != 0 {
			return nil, fmt.Errorf("%w: argo_asf block", avutil.ErrInvalidData)
		}
		data = append(data, pkt.Data...)
	}
	name := strings.TrimSuffix(filepath.Base(url), filepath.Ext(url))
	if len(name) > 8 {
		name = name[:8]
	}
	body := make([]byte, 44+len(data))
	copy(body[0:], "ASF\x00")
	binary.LittleEndian.PutUint16(body[4:], 2)
	binary.LittleEndian.PutUint16(body[6:], 1)
	binary.LittleEndian.PutUint32(body[8:], 1)
	binary.LittleEndian.PutUint32(body[12:], 24)
	copy(body[16:], name)
	binary.LittleEndian.PutUint32(body[24:], uint32(len(data)/17))
	binary.LittleEndian.PutUint32(body[28:], 32)
	binary.LittleEndian.PutUint16(body[36:], uint16(rate))
	binary.LittleEndian.PutUint16(body[38:], 0xFFFF)
	binary.LittleEndian.PutUint32(body[40:], 0x0D)
	copy(body[44:], data)
	return body, nil
}
