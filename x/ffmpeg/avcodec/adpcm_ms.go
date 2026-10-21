package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// msAdapt is ffmpeg's MS ADPCM adaptation table.
var msAdapt = [16]int{
	230, 230, 230, 230, 307, 409, 512, 614,
	768, 614, 512, 409, 307, 230, 230, 230,
}

// msCoeff1 and msCoeff2 are ffmpeg's MS ADPCM coefficients, already divided by 4.
var msCoeff1 = [7]int{64, 128, 0, 48, 60, 115, 98}
var msCoeff2 = [7]int{0, -64, 0, 16, 0, -52, -58}

// msCodec encodes and decodes mono MS ADPCM at ffmpeg's default 1024-byte block.
type msCodec struct {
	pcm    []int16
	idelta int
	rate   int
}

func newMSADPCM(par Parameters) *msCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &msCodec{idelta: 16, rate: rate}
}

func (c *msCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ms adpcm", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ms adpcm samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitMS(false)
}

func (c *msCodec) Flush() ([]*avutil.Packet, error) { return c.emitMS(true) }

func (c *msCodec) emitMS(pad bool) ([]*avutil.Packet, error) {
	const block = 1024
	frame := (block-7)*2 + 2
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
		raw, idelta, err := msBlock(chunk, block, c.idelta)
		if err != nil {
			return nil, err
		}
		c.idelta = idelta
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(frame)})
	}
	if pad {
		c.pcm = nil
	}
	return pkts, nil
}

// MSADPCMEncode packs one mono Microsoft ADPCM block.
// blockSize is ffmpeg's -block_size. The sample count is (blockSize-7)*2+2.
func MSADPCMEncode(pcm []int16, blockSize int) ([]byte, error) {
	raw, _, err := msBlock(pcm, blockSize, 16)
	return raw, err
}

func msBlock(pcm []int16, blockSize, idelta int) ([]byte, int, error) {
	if blockSize < 32 || blockSize&(blockSize-1) != 0 {
		return nil, idelta, fmt.Errorf("%w: ms adpcm block", avutil.ErrInvalidData)
	}
	frame := (blockSize-7)*2 + 2
	if len(pcm) != frame {
		return nil, idelta, fmt.Errorf("%w: ms adpcm samples", avutil.ErrInvalidData)
	}
	if idelta < 16 {
		idelta = 16
	}
	out := make([]byte, blockSize)
	// Predictor 0. coeff1 is 64 and coeff2 is 0, so the predictor is sample1.
	out[0] = 0
	binary.LittleEndian.PutUint16(out[1:], uint16(idelta))
	sample2 := int(pcm[0])
	sample1 := int(pcm[1])
	binary.LittleEndian.PutUint16(out[3:], uint16(int16(sample1)))
	binary.LittleEndian.PutUint16(out[5:], uint16(int16(sample2)))
	pos := 2
	for i := 7; i < blockSize; i++ {
		var hi, lo byte
		hi, sample1, sample2, idelta = msCompress(sample1, sample2, idelta, pcm[pos])
		pos++
		lo, sample1, sample2, idelta = msCompress(sample1, sample2, idelta, pcm[pos])
		pos++
		out[i] = hi<<4 | lo
	}
	return out, idelta, nil
}

func (c *msCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples, err := MSADPCMDecode(pkt.Data, len(pkt.Data))
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// MSADPCMDecode expands mono Microsoft ADPCM blocks. blockSize is the WAV
// block align. The predictor byte selects the coefficient pair.
func MSADPCMDecode(data []byte, blockSize int) ([]int16, error) {
	if blockSize < 7 {
		return nil, fmt.Errorf("%w: ms adpcm block", avutil.ErrInvalidData)
	}
	var out []int16
	for len(data) > 0 {
		n := blockSize
		if n > len(data) {
			n = len(data)
		}
		samples, err := msWAVBlock(data[:n])
		if err != nil {
			return nil, err
		}
		out = append(out, samples...)
		data = data[n:]
	}
	return out, nil
}

func msWAVBlock(block []byte) ([]int16, error) {
	if len(block) < 7 {
		return nil, fmt.Errorf("%w: ms adpcm block", avutil.ErrInvalidData)
	}
	which := int(block[0])
	if which > 6 {
		return nil, fmt.Errorf("%w: ms adpcm predictor %d", avutil.ErrInvalidData, which)
	}
	coeff1 := msCoeff1[which]
	coeff2 := msCoeff2[which]
	idelta := int(int16(binary.LittleEndian.Uint16(block[1:])))
	sample1 := int(int16(binary.LittleEndian.Uint16(block[3:])))
	sample2 := int(int16(binary.LittleEndian.Uint16(block[5:])))
	out := make([]int16, 2+(len(block)-7)*2)
	out[0] = int16(sample2)
	out[1] = int16(sample1)
	o := 2
	for _, b := range block[7:] {
		out[o] = msExpand(&sample1, &sample2, &idelta, coeff1, coeff2, int(b)>>4)
		out[o+1] = msExpand(&sample1, &sample2, &idelta, coeff1, coeff2, int(b)&0x0F)
		o += 2
	}
	return out, nil
}

func msExpand(sample1, sample2, idelta *int, coeff1, coeff2, nibble int) int16 {
	predictor := ((*sample1)*coeff1 + (*sample2)*coeff2) / 64
	delta := nibble
	if nibble&0x08 != 0 {
		delta = nibble - 0x10
	}
	predictor += delta * (*idelta)
	*sample2 = *sample1
	*sample1 = clipInt16(predictor)
	*idelta = (msAdapt[nibble] * (*idelta)) >> 8
	if *idelta < 16 {
		*idelta = 16
	}
	if *idelta > 2147483647/768 {
		*idelta = 2147483647 / 768
	}
	return int16(*sample1)
}

func msCompress(sample1, sample2, idelta int, sample int16) (byte, int, int, int) {
	predictor := sample1 // coeff1/64 with coeff2 == 0
	nibble := int(sample) - predictor
	bias := idelta / 2
	if nibble < 0 {
		bias = -bias
	}
	nibble = (nibble + bias) / idelta
	if nibble > 7 {
		nibble = 7
	} else if nibble < -8 {
		nibble = -8
	}
	nibble &= 0x0f
	delta := nibble
	if nibble&0x08 != 0 {
		delta = nibble - 0x10
	}
	predictor += delta * idelta
	if predictor > 32767 {
		predictor = 32767
	} else if predictor < -32768 {
		predictor = -32768
	}
	sample2 = sample1
	sample1 = predictor
	idelta = (msAdapt[nibble] * idelta) >> 8
	if idelta < 16 {
		idelta = 16
	}
	return byte(nibble), sample1, sample2, idelta
}
