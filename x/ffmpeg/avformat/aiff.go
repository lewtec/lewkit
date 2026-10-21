package avformat

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func writeAIFF(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMIMAQT {
		return nil, fmt.Errorf("%w: aiff", avutil.ErrUnsupported)
	}
	if streams[0].Par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: aiff channels", avutil.ErrInvalidData)
	}
	var pcm []byte
	for _, pkt := range pkts {
		if pkt != nil {
			pcm = append(pcm, pkt.Data...)
		}
	}
	rate := streams[0].Par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	const block = 34
	comm := make([]byte, 24)
	binary.BigEndian.PutUint16(comm[0:], 1)
	binary.BigEndian.PutUint32(comm[2:], uint32(len(pcm)/block))
	binary.BigEndian.PutUint16(comm[6:], 4)
	copy(comm[8:], aiffExtRate(rate))
	copy(comm[18:], "ima4")

	ssndSize := 8 + len(pcm)
	body := make([]byte, 0, 72+len(pcm))
	body = append(body, "FORM"...)
	body = appendU32(body, uint32(64+len(pcm)))
	body = append(body, "AIFC"...)
	body = append(body, "FVER"...)
	body = appendU32(body, 4)
	body = appendU32(body, 0xA2805140)
	body = append(body, "COMM"...)
	body = appendU32(body, 24)
	body = append(body, comm...)
	body = append(body, "SSND"...)
	body = appendU32(body, uint32(ssndSize))
	body = appendU32(body, 0)
	body = appendU32(body, 0)
	body = append(body, pcm...)
	return body, nil
}

func aiffExtRate(rate int) []byte {
	bits := math.Float64bits(float64(rate))
	exp := uint16((bits >> 52) + (16383 - 1023))
	mant := uint64(1)<<63 | (bits << 11)
	var b [10]byte
	binary.BigEndian.PutUint16(b[0:2], exp)
	binary.BigEndian.PutUint64(b[2:10], mant)
	return b[:]
}

func openAIFF(b []byte) (Demuxer, error) {
	if len(b) < 12 || string(b[0:4]) != "FORM" {
		return nil, fmt.Errorf("%w: aiff", avutil.ErrInvalidData)
	}
	kind := string(b[8:12])
	if kind != "AIFF" && kind != "AIFC" {
		return nil, fmt.Errorf("%w: aiff %s", avutil.ErrInvalidData, kind)
	}
	var channels, rate int
	var comp string
	var data []byte
	for off := 12; off+8 <= len(b); {
		id := string(b[off : off+4])
		sz := int(binary.BigEndian.Uint32(b[off+4 : off+8]))
		off += 8
		if sz < 0 || off+sz > len(b) {
			return nil, fmt.Errorf("%w: aiff chunk", avutil.ErrInvalidData)
		}
		chunk := b[off : off+sz]
		switch id {
		case "COMM":
			if len(chunk) < 18 {
				return nil, fmt.Errorf("%w: aiff comm", avutil.ErrInvalidData)
			}
			channels = int(binary.BigEndian.Uint16(chunk[0:2]))
			rate = aiffExtToInt(chunk[8:18])
			if len(chunk) >= 22 {
				comp = string(chunk[18:22])
			}
		case "SSND":
			if len(chunk) < 8 {
				return nil, fmt.Errorf("%w: aiff ssnd", avutil.ErrInvalidData)
			}
			skip := int(binary.BigEndian.Uint32(chunk[0:4]))
			payload := chunk[8:]
			if skip < 0 || skip > len(payload) {
				return nil, fmt.Errorf("%w: aiff ssnd", avutil.ErrInvalidData)
			}
			data = append([]byte(nil), payload[skip:]...)
		}
		off += sz
		if sz%2 == 1 {
			off++
		}
	}
	if comp != "ima4" {
		return nil, fmt.Errorf("%w: aiff %s", avutil.ErrUnsupported, comp)
	}
	if rate <= 0 {
		rate = 44100
	}
	if channels <= 0 {
		channels = 1
	}
	tb := avutil.Q(1, int32(rate))
	pkts := []*avutil.Packet{{Data: data, TimeBase: tb, Key: true}}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: avcodec.IDADPCMIMAQT, SampleFormat: avutil.SampleS16,
			SampleRate: rate, Layout: layoutOr(avutil.ChannelLayout{}, channels), TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

func aiffExtToInt(b []byte) int {
	exp := int(binary.BigEndian.Uint16(b[:2]))
	mant := binary.BigEndian.Uint64(b[2:10])
	if mant == 0 || exp == 0 {
		return 0
	}
	e := exp - 16383
	if e < 0 || e > 63 {
		return 0
	}
	return int(mant >> (63 - e))
}

func appendU32(b []byte, v uint32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	return append(b, buf[:]...)
}
