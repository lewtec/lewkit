package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openWAV(b []byte) (Demuxer, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, fmt.Errorf("%w: wav", avutil.ErrInvalidData)
	}
	var tag uint16
	var channels, rate, bits int
	var data []byte
	for i := 12; i+8 <= len(b); {
		id := string(b[i : i+4])
		sz := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		i += 8
		if sz < 0 || i+sz > len(b) {
			return nil, fmt.Errorf("%w: wav chunk", avutil.ErrInvalidData)
		}
		chunk := b[i : i+sz]
		switch id {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, fmt.Errorf("%w: wav fmt", avutil.ErrInvalidData)
			}
			tag = binary.LittleEndian.Uint16(chunk[0:2])
			channels = int(binary.LittleEndian.Uint16(chunk[2:4]))
			rate = int(binary.LittleEndian.Uint32(chunk[4:8]))
			bits = int(binary.LittleEndian.Uint16(chunk[14:16]))
		case "data":
			data = append([]byte(nil), chunk...)
		}
		i += sz
		if sz%2 == 1 {
			i++
		}
	}
	format, id := wavCodec(tag, bits)
	if id == 0 {
		return nil, fmt.Errorf("%w: wav format %d", avutil.ErrUnsupported, tag)
	}
	return openRawAudio(format, data, Options{SampleRate: rate, Layout: layoutOr(avutil.ChannelLayout{}, channels)})
}

func wavCodec(tag uint16, bits int) (string, avcodec.ID) {
	switch tag {
	case 1:
		switch bits {
		case 8:
			return "u8", avcodec.IDPCMU8
		case 16:
			return "s16le", avcodec.IDPCMS16LE
		case 24:
			return "s24le", avcodec.IDPCMS24LE
		case 32:
			return "s32le", avcodec.IDPCMS32LE
		}
	case 3:
		if bits == 32 {
			return "f32le", avcodec.IDPCMF32LE
		}
	case 6:
		return "alaw", avcodec.IDPCMALAW
	case 7:
		return "mulaw", avcodec.IDPCMMULAW
	}
	return "", 0
}

func writeWAV(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: wav stream", avutil.ErrInvalidData)
	}
	par := streams[0].Par
	var pcm []byte
	for _, pkt := range pkts {
		if pkt.Stream != 0 && len(streams) > 1 {
			continue
		}
		pcm = append(pcm, pkt.Data...)
	}
	tag := uint16(1)
	bits := 16
	channels := par.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	block := 0
	fmtExtra := []byte(nil)
	switch par.ID {
	case avcodec.IDPCMU8:
		bits = 8
	case avcodec.IDPCMS16LE:
		bits = 16
	case avcodec.IDPCMS24LE:
		bits = 24
	case avcodec.IDPCMS32LE:
		bits = 32
	case avcodec.IDPCMF32LE:
		tag, bits = 3, 32
	case avcodec.IDPCMALAW:
		tag, bits = 6, 8
	case avcodec.IDPCMMULAW:
		tag, bits = 7, 8
	case avcodec.IDGSMMS:
		tag = 0x31
		bits = 0
		block = 65
		fmtExtra = []byte{2, 0, 0x40, 0x01}
	case avcodec.IDADPCMIMAWAV:
		tag = 0x11
		bits = 4
		block = 1024
		fmtExtra = []byte{2, 0, 0xf9, 0x07} // 2041 samples per block
	case avcodec.IDADPCMMS:
		tag = 0x02
		bits = 4
		block = 1024
		fmtExtra = msWAVExtra()
	case avcodec.IDADPCMYAMAHA:
		tag = 0x20
		bits = 4
		block = 1024
		fmtExtra = []byte{0, 0}
	default:
		return nil, fmt.Errorf("%w: wav %s", avutil.ErrUnsupported, par.ID)
	}
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	if block == 0 {
		block = channels * bits / 8
	}
	byteRate := rate * block
	switch par.ID {
	case avcodec.IDGSMMS:
		byteRate = rate * 65 / 320
	case avcodec.IDADPCMIMAWAV:
		byteRate = rate * 1024 / 2041
	case avcodec.IDADPCMMS:
		byteRate = rate * 1024 / 2036
	case avcodec.IDADPCMYAMAHA:
		byteRate = rate / 2
	}
	fmtSize := 16 + len(fmtExtra)
	hdr := make([]byte, 28+fmtSize)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(20+fmtSize+len(pcm)))
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], uint32(fmtSize))
	binary.LittleEndian.PutUint16(hdr[20:], tag)
	binary.LittleEndian.PutUint16(hdr[22:], uint16(channels))
	binary.LittleEndian.PutUint32(hdr[24:], uint32(rate))
	binary.LittleEndian.PutUint32(hdr[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(hdr[32:], uint16(block))
	binary.LittleEndian.PutUint16(hdr[34:], uint16(bits))
	copy(hdr[36:], fmtExtra)
	dataAt := 20 + fmtSize
	copy(hdr[dataAt:], "data")
	binary.LittleEndian.PutUint32(hdr[dataAt+4:], uint32(len(pcm)))
	return append(hdr, pcm...), nil
}

// msWAVExtra is the WAVEFORMATEX tail for Microsoft ADPCM: cbSize 32,
// 2036 samples per block, and the seven coefficient pairs ffmpeg writes.
func msWAVExtra() []byte {
	coeffs := []int16{256, 0, 512, -256, 0, 0, 192, 64, 240, 0, 460, -208, 392, -232}
	extra := []byte{32, 0, 0xf4, 0x07, 7, 0}
	for _, c := range coeffs {
		extra = append(extra, byte(c), byte(uint16(c)>>8))
	}
	return extra
}
