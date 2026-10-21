package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openAUD(b []byte) (Demuxer, error) {
	if len(b) < 12 || b[11] != 99 {
		return nil, fmt.Errorf("%w: wsaud", avutil.ErrInvalidData)
	}
	rate := int(binary.LittleEndian.Uint16(b[0:2]))
	if rate <= 0 {
		rate = 44100
	}
	tb := avutil.Q(1, int32(rate))
	var pkts []*avutil.Packet
	var pts int64
	for off := 12; off+8 <= len(b); {
		n := int(binary.LittleEndian.Uint16(b[off:]))
		sig := binary.LittleEndian.Uint32(b[off+4:])
		if sig != 0x0000DEAF {
			return nil, fmt.Errorf("%w: wsaud signature", avutil.ErrInvalidData)
		}
		off += 8
		if n < 0 || off+n > len(b) {
			return nil, fmt.Errorf("%w: wsaud chunk", avutil.ErrInvalidData)
		}
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), b[off:off+n]...),
			PTS:  pts, DTS: pts, Duration: int64(n * 2), TimeBase: tb, Key: true,
		})
		pts += int64(n * 2)
		off += n
	}
	if len(pkts) == 0 {
		pkts = []*avutil.Packet{{TimeBase: tb, Key: true}}
	}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDADPCMIMAWS, SampleFormat: avutil.SampleS16,
			SampleRate: rate, Layout: avutil.Mono(), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

func writeAUD(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMIMAWS {
		return nil, fmt.Errorf("%w: wsaud", avutil.ErrUnsupported)
	}
	if streams[0].Par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: wsaud channels", avutil.ErrInvalidData)
	}
	rate := streams[0].Par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	var chunks []byte
	size := 0
	uncomp := 0
	for _, pkt := range pkts {
		if pkt == nil || len(pkt.Data) == 0 {
			continue
		}
		n := len(pkt.Data)
		var hdr [8]byte
		binary.LittleEndian.PutUint16(hdr[0:], uint16(n))
		binary.LittleEndian.PutUint16(hdr[2:], uint16(n*4))
		binary.LittleEndian.PutUint32(hdr[4:], 0x0000DEAF)
		chunks = append(chunks, hdr[:]...)
		chunks = append(chunks, pkt.Data...)
		size += n + 8
		uncomp += n * 4
	}
	body := make([]byte, 12+len(chunks))
	binary.LittleEndian.PutUint16(body[0:], uint16(rate))
	binary.LittleEndian.PutUint32(body[2:], uint32(size))
	binary.LittleEndian.PutUint32(body[6:], uint32(uncomp))
	body[10] = 2
	body[11] = 99
	copy(body[12:], chunks)
	return body, nil
}
