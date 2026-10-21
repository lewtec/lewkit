package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// imaIndex and imaStep are the IMA ADPCM tables ffmpeg uses.
var imaIndex = [16]int8{
	-1, -1, -1, -1, 2, 4, 6, 8,
	-1, -1, -1, -1, 2, 4, 6, 8,
}

var imaStep = [89]int16{
	7, 8, 9, 10, 11, 12, 13, 14, 16, 17,
	19, 21, 23, 25, 28, 31, 34, 37, 41, 45,
	50, 55, 60, 66, 73, 80, 88, 97, 107, 118,
	130, 143, 157, 173, 190, 209, 230, 253, 279, 307,
	337, 371, 408, 449, 494, 544, 598, 658, 724, 796,
	876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066,
	2272, 2499, 2749, 3024, 3327, 3660, 4026, 4428, 4871, 5358,
	5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
	15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767,
}

var imaDiff = [16]int8{
	1, 3, 5, 7, 9, 11, 13, 15,
	-1, -3, -5, -7, -9, -11, -13, -15,
}

// imaCodec encodes and decodes mono IMA WAV ADPCM at ffmpeg's default 1024-byte block.
type imaCodec struct {
	pcm  []int16
	step int
	rate int
}

func newIMAWAV(par Parameters) *imaCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &imaCodec{rate: rate}
}

func (c *imaCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima wav", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima wav samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitIMA(false)
}

func (c *imaCodec) Flush() ([]*avutil.Packet, error) {
	return c.emitIMA(true)
}

func (c *imaCodec) emitIMA(pad bool) ([]*avutil.Packet, error) {
	const block = 1024
	frame := (block-4)*2 + 1
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
		raw, step, err := imaBlock(chunk, block, c.step)
		if err != nil {
			return nil, err
		}
		c.step = step
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(frame)})
	}
	if pad {
		c.pcm = nil
	}
	return pkts, nil
}

// IMAWAVEncode packs mono 16-bit PCM into one IMA WAV ADPCM block.
// blockSize is ffmpeg's -block_size (a power of two, at least 32).
// The sample count must be (blockSize-4)*2+1, which is one frame.
func IMAWAVEncode(pcm []int16, blockSize int) ([]byte, error) {
	raw, _, err := imaBlock(pcm, blockSize, 0)
	return raw, err
}

func imaBlock(pcm []int16, blockSize, step int) ([]byte, int, error) {
	if blockSize < 32 || blockSize&(blockSize-1) != 0 {
		return nil, step, fmt.Errorf("%w: ima wav block", avutil.ErrInvalidData)
	}
	frame := (blockSize-4)*2 + 1
	if len(pcm) != frame {
		return nil, step, fmt.Errorf("%w: ima wav samples", avutil.ErrInvalidData)
	}
	out := make([]byte, blockSize)
	binary.LittleEndian.PutUint16(out[0:], uint16(pcm[0]))
	out[2] = byte(step)
	prev := int(pcm[0])
	dst := 4
	blocks := (len(pcm) - 1) / 8
	for i := 0; i < blocks; i++ {
		smp := pcm[1+i*8:]
		for j := 0; j < 8; j += 2 {
			lo := imaCompress(&prev, &step, smp[j])
			hi := imaCompress(&prev, &step, smp[j+1])
			out[dst] = lo | hi<<4
			dst++
		}
	}
	return out, step, nil
}

func (c *imaCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples, err := IMAWAVDecode(pkt.Data, len(pkt.Data))
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// IMAWAVDecode expands mono 4-bit IMA WAV blocks. blockSize is the WAV
// block align. Each block starts with a little-endian predictor and step.
func IMAWAVDecode(data []byte, blockSize int) ([]int16, error) {
	if blockSize < 4 {
		return nil, fmt.Errorf("%w: ima wav block", avutil.ErrInvalidData)
	}
	var out []int16
	for len(data) > 0 {
		n := blockSize
		if n > len(data) {
			n = len(data)
		}
		samples, err := imaWAVBlock(data[:n])
		if err != nil {
			return nil, err
		}
		out = append(out, samples...)
		data = data[n:]
	}
	return out, nil
}

func imaWAVBlock(block []byte) ([]int16, error) {
	if len(block) < 4 {
		return nil, fmt.Errorf("%w: ima wav block", avutil.ErrInvalidData)
	}
	pred := int(int16(binary.LittleEndian.Uint16(block)))
	step := int(int16(binary.LittleEndian.Uint16(block[2:])))
	if step < 0 || step > 88 {
		return nil, fmt.Errorf("%w: ima wav step %d", avutil.ErrInvalidData, step)
	}
	groups := (len(block) - 4) / 4
	out := make([]int16, 1+groups*8)
	out[0] = int16(pred)
	o := 1
	for src := 4; src < 4+groups*4; src++ {
		v := int(block[src])
		out[o] = imaExpand(&pred, &step, v&0x0F)
		out[o+1] = imaExpand(&pred, &step, v>>4)
		o += 2
	}
	return out, nil
}

func imaExpand(pred, step *int, nibble int) int16 {
	st := int(imaStep[*step])
	idx := *step + int(imaIndex[nibble])
	if idx < 0 {
		idx = 0
	} else if idx > 88 {
		idx = 88
	}
	diff := ((2*(nibble&7) + 1) * st) >> 3
	p := *pred
	if nibble&8 != 0 {
		p -= diff
	} else {
		p += diff
	}
	*pred = clipInt16(p)
	*step = idx
	return int16(*pred)
}

func clipInt16(v int) int {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}

func pcmS16Mono(samples []int16, rate int, pts int64) (*avutil.Frame, error) {
	if len(samples) == 0 {
		return nil, nil
	}
	if rate <= 0 {
		rate = 44100
	}
	out, err := avutil.NewAudio(avutil.SampleS16, rate, avutil.Mono(), len(samples))
	if err != nil {
		return nil, err
	}
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out.Data[0][i*2:], uint16(s))
	}
	out.PTS = pts
	out.Duration = int64(len(samples))
	out.TimeBase = avutil.Q(1, int32(rate))
	return out, nil
}

func imaCompress(prev, step *int, sample int16) byte {
	delta := int(sample) - *prev
	mag := delta
	if mag < 0 {
		mag = -mag
	}
	nibble := mag * 4 / int(imaStep[*step])
	if nibble > 7 {
		nibble = 7
	}
	if delta < 0 {
		nibble += 8
	}
	*prev += int(imaStep[*step]) * int(imaDiff[nibble]) / 8
	if *prev > 32767 {
		*prev = 32767
	} else if *prev < -32768 {
		*prev = -32768
	}
	*step += int(imaIndex[nibble])
	if *step < 0 {
		*step = 0
	} else if *step > 88 {
		*step = 88
	}
	return byte(nibble)
}
