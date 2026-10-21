package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// ssiCodec encodes mono IMA Simon & Schuster ADPCM.
// A full frame is 2048 samples (block size 1024). Predictor and step
// carry across frames. There is no block header. A trailing odd sample
// is dropped; ffmpeg's bit writer asserts on that last nibble.
type ssiCodec struct {
	pcm  []int16
	prev int
	step int
}

func newIMASSI(Parameters) *ssiCodec { return &ssiCodec{} }

func (c *ssiCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima ssi", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima ssi samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitSSI(false)
}

func (c *ssiCodec) Flush() ([]*avutil.Packet, error) { return c.emitSSI(true) }

func (c *ssiCodec) emitSSI(flush bool) ([]*avutil.Packet, error) {
	const frame = 2048
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (flush && len(c.pcm) >= 2) {
		n := frame
		if n > len(c.pcm) {
			n = len(c.pcm) - len(c.pcm)%2
		}
		raw, prev, step := ssiPacket(c.pcm[:n], c.prev, c.step)
		c.prev, c.step = prev, step
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(n)})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

// IMASSIEncode packs mono PCM into IMA SSI ADPCM.
// Full packets hold 2048 samples. A short even tail is kept.
// Nibbles are stored high-first, using the QuickTime IMA compressor.
func IMASSIEncode(pcm []int16) []byte {
	const frame = 2048
	var out []byte
	prev, step := 0, 0
	for len(pcm) >= 2 {
		n := frame
		if n > len(pcm) {
			n = len(pcm) - len(pcm)%2
		}
		raw, p, s := ssiPacket(pcm[:n], prev, step)
		prev, step = p, s
		out = append(out, raw...)
		pcm = pcm[n:]
	}
	return out
}

func ssiPacket(pcm []int16, prev, step int) ([]byte, int, int) {
	out := make([]byte, len(pcm)/2)
	for i := 0; i < len(out); i++ {
		hi := imaQTCompress(&prev, &step, pcm[i*2])
		lo := imaQTCompress(&prev, &step, pcm[i*2+1])
		out[i] = hi<<4 | lo
	}
	return out, prev, step
}
