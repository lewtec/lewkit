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

func appendU32(b []byte, v uint32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	return append(b, buf[:]...)
}
