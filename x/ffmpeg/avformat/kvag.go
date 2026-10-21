package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

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
