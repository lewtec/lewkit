package avformat

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// writeALP builds a LEGO Racers ALP file. A .pcm path stores the sample
// rate (header size 12). Any other path is a .tun file at 22050 Hz.
func writeALP(url string, streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) != 1 || streams[0].Par.ID != avcodec.IDADPCMIMAALP {
		return nil, fmt.Errorf("%w: alp", avutil.ErrUnsupported)
	}
	channels := streams[0].Par.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	if channels > 1 {
		return nil, fmt.Errorf("%w: alp channels", avutil.ErrInvalidData)
	}
	rate := streams[0].Par.SampleRate
	if rate <= 0 {
		rate = 22050
	}
	pcm := strings.EqualFold(filepath.Ext(url), ".pcm")
	if pcm {
		if rate > 44100 {
			return nil, fmt.Errorf("%w: alp rate %d", avutil.ErrInvalidData, rate)
		}
	} else if rate != 22050 {
		return nil, fmt.Errorf("%w: alp tun rate %d", avutil.ErrInvalidData, rate)
	}
	var data []byte
	for _, pkt := range pkts {
		if pkt == nil || len(pkt.Data) == 0 {
			continue
		}
		data = append(data, pkt.Data...)
	}
	hdr := 16
	if pcm {
		hdr = 20
	}
	body := make([]byte, hdr+len(data))
	copy(body[0:], "ALP ")
	if pcm {
		binary.LittleEndian.PutUint32(body[4:], 12)
	} else {
		binary.LittleEndian.PutUint32(body[4:], 8)
	}
	copy(body[8:], "ADPCM")
	body[15] = byte(channels)
	if pcm {
		binary.LittleEndian.PutUint32(body[16:], uint32(rate))
	}
	copy(body[hdr:], data)
	return body, nil
}
