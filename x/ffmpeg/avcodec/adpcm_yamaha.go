package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// yamahaScale is ffmpeg's Yamaha ADPCM step scale.
var yamahaScale = [16]int{
	230, 230, 230, 230, 307, 409, 512, 614,
	230, 230, 230, 230, 307, 409, 512, 614,
}

// yamahaCodec encodes and decodes mono Yamaha ADPCM.
// Predictor and step carry across packets. There is no block header.
type yamahaCodec struct {
	pcm  []int16
	pred int
	step int
	rate int
}

func newYamaha(par Parameters) *yamahaCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &yamahaCodec{rate: rate}
}

func (c *yamahaCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: yamaha", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: yamaha samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitYamaha(false)
}

func (c *yamahaCodec) Flush() ([]*avutil.Packet, error) { return c.emitYamaha(true) }

func (c *yamahaCodec) emitYamaha(pad bool) ([]*avutil.Packet, error) {
	const block = 1024
	frame := block * 2
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (pad && len(c.pcm) > 0) {
		chunk := c.pcm
		if len(chunk) < frame {
			chunk = make([]int16, frame)
			copy(chunk, c.pcm)
			c.pcm = nil
		} else {
			c.pcm = append([]int16(nil), c.pcm[frame:]...)
		}
		raw, pred, step, err := yamahaBlock(chunk, block, c.pred, c.step)
		if err != nil {
			return nil, err
		}
		c.pred, c.step = pred, step
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(frame)})
	}
	if pad {
		c.pcm = nil
	}
	return pkts, nil
}

func (c *yamahaCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples := yamahaExpandBytes(pkt.Data, &c.pred, &c.step)
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// YamahaDecode expands a mono Yamaha ADPCM stream from a reset predictor.
func YamahaDecode(data []byte) []int16 {
	pred, step := 0, 0
	return yamahaExpandBytes(data, &pred, &step)
}

func yamahaExpandBytes(data []byte, pred, step *int) []int16 {
	out := make([]int16, len(data)*2)
	for i, b := range data {
		out[i*2] = yamahaExpand(pred, step, int(b)&0x0F)
		out[i*2+1] = yamahaExpand(pred, step, int(b)>>4)
	}
	return out
}

func yamahaExpand(pred, step *int, nibble int) int16 {
	if *step == 0 {
		*pred = 0
		*step = 127
	}
	*pred = clipInt16(*pred + (*step*int(imaDiff[nibble]))/8)
	*step = (*step * yamahaScale[nibble]) >> 8
	if *step < 127 {
		*step = 127
	} else if *step > 24576 {
		*step = 24576
	}
	return int16(*pred)
}

// YamahaEncode packs mono PCM into one Yamaha ADPCM block.
// blockSize is ffmpeg's -block_size. The sample count is blockSize*2.
func YamahaEncode(pcm []int16, blockSize int) ([]byte, error) {
	raw, _, _, err := yamahaBlock(pcm, blockSize, 0, 0)
	return raw, err
}

func yamahaBlock(pcm []int16, blockSize, pred, step int) ([]byte, int, int, error) {
	if blockSize < 32 || blockSize&(blockSize-1) != 0 {
		return nil, pred, step, fmt.Errorf("%w: yamaha block", avutil.ErrInvalidData)
	}
	if len(pcm) != blockSize*2 {
		return nil, pred, step, fmt.Errorf("%w: yamaha samples", avutil.ErrInvalidData)
	}
	out := make([]byte, blockSize)
	for i := 0; i < blockSize; i++ {
		lo := yamahaCompress(&pred, &step, pcm[i*2])
		hi := yamahaCompress(&pred, &step, pcm[i*2+1])
		out[i] = lo | hi<<4
	}
	return out, pred, step, nil
}

func yamahaCompress(pred, step *int, sample int16) byte {
	if *step == 0 {
		*pred = 0
		*step = 127
	}
	delta := int(sample) - *pred
	mag := delta
	if mag < 0 {
		mag = -mag
	}
	nibble := mag * 4 / *step
	if nibble > 7 {
		nibble = 7
	}
	if delta < 0 {
		nibble += 8
	}
	*pred += *step * int(imaDiff[nibble]) / 8
	if *pred > 32767 {
		*pred = 32767
	} else if *pred < -32768 {
		*pred = -32768
	}
	*step = (*step * yamahaScale[nibble]) >> 8
	if *step < 127 {
		*step = 127
	} else if *step > 24576 {
		*step = 24576
	}
	return byte(nibble)
}
