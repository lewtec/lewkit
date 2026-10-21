package swresample

import (
	"encoding/binary"
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func loadS32(format avutil.SampleFormat, b []byte) int32 {
	switch avutil.PackedSample(format) {
	case avutil.SampleU8:
		return int32(int(b[0])-128) << 24
	case avutil.SampleS16:
		return int32(int16(binary.LittleEndian.Uint16(b))) << 16
	case avutil.SampleS32:
		return int32(binary.LittleEndian.Uint32(b))
	case avutil.SampleS64:
		return int32(int64(binary.LittleEndian.Uint64(b)) >> 32)
	default:
		return 0
	}
}

func storeS32(format avutil.SampleFormat, b []byte, v int32) {
	switch avutil.PackedSample(format) {
	case avutil.SampleU8:
		b[0] = byte(int(v>>24) + 128)
	case avutil.SampleS16:
		binary.LittleEndian.PutUint16(b, uint16(int16(v>>16)))
	case avutil.SampleS32:
		binary.LittleEndian.PutUint32(b, uint32(v))
	case avutil.SampleS64:
		binary.LittleEndian.PutUint64(b, uint64(int64(v)<<32))
	}
}

func loadFloat(format avutil.SampleFormat, b []byte) float64 {
	switch avutil.PackedSample(format) {
	case avutil.SampleU8:
		return float64(int(b[0])-128) * (1.0 / 128)
	case avutil.SampleS16:
		return float64(int16(binary.LittleEndian.Uint16(b))) * (1.0 / 32768)
	case avutil.SampleS32:
		return float64(int32(binary.LittleEndian.Uint32(b))) * (1.0 / 2147483648)
	case avutil.SampleS64:
		return float64(int64(binary.LittleEndian.Uint64(b))) * (1.0 / 9223372036854775808.0)
	case avutil.SampleFLT:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
	case avutil.SampleDBL:
		return math.Float64frombits(binary.LittleEndian.Uint64(b))
	default:
		return 0
	}
}

func storeFloat(format avutil.SampleFormat, b []byte, v float64) {
	switch avutil.PackedSample(format) {
	case avutil.SampleU8:
		b[0] = clipU8(int(math.RoundToEven(v*128)) + 128)
	case avutil.SampleS16:
		binary.LittleEndian.PutUint16(b, uint16(clipS16(int(math.RoundToEven(v*32768)))))
	case avutil.SampleS32:
		rounded := math.RoundToEven(v * 2147483648)
		binary.LittleEndian.PutUint32(b, uint32(clipS32(rounded)))
	case avutil.SampleS64:
		rounded := math.RoundToEven(v * 9223372036854775808.0)
		binary.LittleEndian.PutUint64(b, uint64(int64(rounded)))
	case avutil.SampleFLT:
		binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v)))
	case avutil.SampleDBL:
		binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	}
}

func clipU8(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

func clipS16(v int) int16 {
	if v < -32768 {
		return -32768
	}
	if v > 32767 {
		return 32767
	}
	return int16(v)
}

func clipS32(v float64) int32 {
	if v > 2147483647 {
		return 2147483647
	}
	if v < -2147483648 {
		return -2147483648
	}
	return int32(v)
}

func isFloat(format avutil.SampleFormat) bool {
	switch avutil.PackedSample(format) {
	case avutil.SampleFLT, avutil.SampleDBL:
		return true
	default:
		return false
	}
}

func plane(frame *avutil.Frame, channel int) []byte {
	if avutil.IsPlanarSample(frame.SampleFmt) {
		return frame.Data[channel]
	}
	return frame.Data[0]
}

func sampleAt(format avutil.SampleFormat, buf []byte, index, channels, channel int) []byte {
	width := avutil.BytesPerSample(format)
	if avutil.IsPlanarSample(format) {
		off := index * width
		return buf[off : off+width]
	}
	off := (index*channels + channel) * width
	return buf[off : off+width]
}

func convertOne(srcFmt, dstFmt avutil.SampleFormat, src, dst []byte) {
	if isFloat(srcFmt) || isFloat(dstFmt) {
		storeFloat(dstFmt, dst, loadFloat(srcFmt, src))
		return
	}
	storeS32(dstFmt, dst, loadS32(srcFmt, src))
}

// s16ToF32 converts interleaved or a single plane of int16 to float32.
// The scale is 1/32768, matching audioconvert.c.
func s16ToF32(dst []float32, src []int16) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	done := 0
	if s16ToF32Asm(dst, src) {
		done = n &^ 7
	}
	const scale = float32(1.0 / 32768.0)
	for i := done; i < n; i++ {
		dst[i] = float32(src[i]) * scale
	}
}

// f32ToS16 converts float32 to int16 with round-to-nearest-even and saturation.
func f32ToS16(dst []int16, src []float32) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	done := 0
	if f32ToS16Asm(dst, src) {
		done = n &^ 7
	}
	for i := done; i < n; i++ {
		dst[i] = clipS16(int(math.RoundToEven(float64(src[i]) * 32768)))
	}
}
