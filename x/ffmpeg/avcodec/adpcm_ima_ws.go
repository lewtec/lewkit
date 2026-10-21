package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// wsCodec encodes mono IMA Westwood ADPCM. A full frame is 2048 samples.
// Predictor and step carry across frames. There is no block header.
type wsCodec struct {
	pcm  []int16
	prev int
	step int
	rate int
}

func newIMAWS(par Parameters) *wsCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &wsCodec{rate: rate}
}

func (c *wsCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima ws", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima ws samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitWS(false)
}

func (c *wsCodec) Flush() ([]*avutil.Packet, error) { return c.emitWS(true) }

func (c *wsCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples := imawsExpand(pkt.Data, &c.prev, &c.step)
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

func (c *wsCodec) emitWS(flush bool) ([]*avutil.Packet, error) {
	const frame = 2048
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (flush && len(c.pcm) >= 2) {
		n := frame
		if n > len(c.pcm) {
			n = len(c.pcm) - len(c.pcm)%2
		}
		chunk := c.pcm[:n]
		raw, prev, step := imawsPacket(chunk, c.prev, c.step)
		c.prev, c.step = prev, step
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(n)})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

// IMAWSEncode packs mono PCM into IMA Westwood ADPCM packets.
// Full packets hold 2048 samples. A short tail is kept, and an odd
// leftover sample is dropped, matching ffmpeg's small last frame.
func IMAWSEncode(pcm []int16) []byte {
	const frame = 2048
	var out []byte
	prev, step := 0, 0
	for len(pcm) >= 2 {
		n := frame
		if n > len(pcm) {
			n = len(pcm) - len(pcm)%2
		}
		raw, p, s := imawsPacket(pcm[:n], prev, step)
		prev, step = p, s
		out = append(out, raw...)
		pcm = pcm[n:]
	}
	return out
}

// IMAWSDecode expands mono IMA Westwood ADPCM. Nibbles are low then high.
// Predictor and step start at zero and carry across the whole buffer.
func IMAWSDecode(data []byte) []int16 {
	prev, step := 0, 0
	return imawsExpand(data, &prev, &step)
}

func imawsExpand(data []byte, prev, step *int) []int16 {
	out := make([]int16, len(data)*2)
	for i, b := range data {
		out[i*2] = imaExpand(prev, step, int(b)&0x0F)
		out[i*2+1] = imaExpand(prev, step, int(b)>>4)
	}
	return out
}

func imawsPacket(pcm []int16, prev, step int) ([]byte, int, int) {
	out := make([]byte, len(pcm)/2)
	for i := 0; i < len(out); i++ {
		lo := imaCompress(&prev, &step, pcm[i*2])
		hi := imaCompress(&prev, &step, pcm[i*2+1])
		out[i] = lo | hi<<4
	}
	return out, prev, step
}
