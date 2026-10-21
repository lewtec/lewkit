package avcodec

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// Snappy compresses and expands the block format used by FFmpeg's HAP decoder
// and by libsnappy. The byte layout follows the Snappy format description:
// a varint length, then tag bytes for literals and back-references.

// SnappyCompress encodes src. The result is a stream libsnappy can expand.
func SnappyCompress(src []byte) []byte {
	var out []byte
	out = appendVarint(out, len(src))
	// Hash table of the last position of each 4-byte fingerprint.
	const hbits = 14
	table := make([]int32, 1<<hbits)
	for i := range table {
		table[i] = -1
	}
	i := 0
	lit := 0
	for i+4 <= len(src) {
		h := (uint32(src[i]) | uint32(src[i+1])<<8 | uint32(src[i+2])<<16 | uint32(src[i+3])<<24) * 0x1e35a7bd
		h >>= 32 - hbits
		prev := int(table[h])
		table[h] = int32(i)
		if prev < 0 || i-prev >= 65536 || i-prev < 1 || !match4(src, prev, i) {
			i++
			continue
		}
		n := 4
		for i+n < len(src) && src[prev+n] == src[i+n] && n < 64 {
			n++
		}
		out = emitLiteral(out, src[lit:i])
		out = emitCopy(out, i-prev, n)
		i += n
		lit = i
	}
	out = emitLiteral(out, src[lit:])
	return out
}

// SnappyUncompress expands a Snappy block into a new buffer.
func SnappyUncompress(src []byte) ([]byte, error) {
	n, k, err := readVarint(src)
	if err != nil {
		return nil, err
	}
	if n < 0 || n > 1<<24 {
		return nil, fmt.Errorf("%w: snappy length", avutil.ErrInvalidData)
	}
	dst := make([]byte, n)
	p := 0
	for k < len(src) {
		if p > n {
			return nil, fmt.Errorf("%w: snappy overrun", avutil.ErrInvalidData)
		}
		tag := src[k]
		k++
		val := int(tag >> 2)
		var wrote int
		switch tag & 3 {
		case 0:
			wrote, k, err = snappyLiteral(src, k, dst[p:], val)
		case 1:
			wrote, k, err = snappyCopy1(src, k, dst, p, val)
		case 2:
			wrote, k, err = snappyCopy2(src, k, dst, p, val)
		case 3:
			wrote, k, err = snappyCopy4(src, k, dst, p, val)
		}
		if err != nil {
			return nil, err
		}
		p += wrote
	}
	if p != n {
		return nil, fmt.Errorf("%w: snappy size", avutil.ErrInvalidData)
	}
	return dst, nil
}

func snappyLiteral(src []byte, k int, dst []byte, val int) (int, int, error) {
	length := 1
	switch val {
	case 63:
		if k+4 > len(src) {
			return 0, k, fmt.Errorf("%w: snappy literal", avutil.ErrInvalidData)
		}
		length += int(src[k]) | int(src[k+1])<<8 | int(src[k+2])<<16 | int(src[k+3])<<24
		k += 4
	case 62:
		if k+3 > len(src) {
			return 0, k, fmt.Errorf("%w: snappy literal", avutil.ErrInvalidData)
		}
		length += int(src[k]) | int(src[k+1])<<8 | int(src[k+2])<<16
		k += 3
	case 61:
		if k+2 > len(src) {
			return 0, k, fmt.Errorf("%w: snappy literal", avutil.ErrInvalidData)
		}
		length += int(src[k]) | int(src[k+1])<<8
		k += 2
	case 60:
		if k >= len(src) {
			return 0, k, fmt.Errorf("%w: snappy literal", avutil.ErrInvalidData)
		}
		length += int(src[k])
		k++
	default:
		length += val
	}
	if length < 0 || length > len(dst) || k+length > len(src) {
		return 0, k, fmt.Errorf("%w: snappy literal", avutil.ErrInvalidData)
	}
	copy(dst[:length], src[k:k+length])
	return length, k + length, nil
}

func snappyCopy1(src []byte, k int, dst []byte, p, val int) (int, int, error) {
	if k >= len(src) {
		return 0, k, fmt.Errorf("%w: snappy copy", avutil.ErrInvalidData)
	}
	length := 4 + (val & 7)
	off := int(src[k]) | (val&0x38)<<5
	k++
	if err := snappyCopy(dst, p, off, length); err != nil {
		return 0, k, err
	}
	return length, k, nil
}

func snappyCopy2(src []byte, k int, dst []byte, p, val int) (int, int, error) {
	if k+2 > len(src) {
		return 0, k, fmt.Errorf("%w: snappy copy", avutil.ErrInvalidData)
	}
	length := 1 + val
	off := int(src[k]) | int(src[k+1])<<8
	k += 2
	if err := snappyCopy(dst, p, off, length); err != nil {
		return 0, k, err
	}
	return length, k, nil
}

func snappyCopy4(src []byte, k int, dst []byte, p, val int) (int, int, error) {
	if k+4 > len(src) {
		return 0, k, fmt.Errorf("%w: snappy copy", avutil.ErrInvalidData)
	}
	length := 1 + val
	off := int(src[k]) | int(src[k+1])<<8 | int(src[k+2])<<16 | int(src[k+3])<<24
	k += 4
	if err := snappyCopy(dst, p, off, length); err != nil {
		return 0, k, err
	}
	return length, k, nil
}

func snappyCopy(dst []byte, p, off, length int) error {
	if off <= 0 || off > p || p+length > len(dst) {
		return fmt.Errorf("%w: snappy copy", avutil.ErrInvalidData)
	}
	if off >= length && snappyCopyFast(dst[p:p+length], dst[p-off:p-off+length]) {
		return nil
	}
	for i := 0; i < length; i++ {
		dst[p+i] = dst[p-off+i]
	}
	return nil
}

func match4(src []byte, a, b int) bool {
	return src[a] == src[b] && src[a+1] == src[b+1] && src[a+2] == src[b+2] && src[a+3] == src[b+3]
}

func emitLiteral(out, lit []byte) []byte {
	for len(lit) > 0 {
		n := len(lit)
		if n > 60 {
			n = 60
		}
		out = append(out, byte((n-1)<<2))
		out = append(out, lit[:n]...)
		lit = lit[n:]
	}
	return out
}

func emitCopy(out []byte, off, n int) []byte {
	for n > 0 {
		chunk := n
		if chunk > 64 {
			chunk = 64
		}
		if chunk < 4 {
			// Too short for a copy tag. Callers only pass n >= 4.
			chunk = n
		}
		if off < 2048 && chunk <= 11 && chunk >= 4 {
			// copy1: len = 4+(val&7), off = byte | (val&0x38)<<5
			val := (chunk - 4) | ((off >> 8) << 3)
			out = append(out, byte(val<<2)|1, byte(off))
		} else {
			// copy2: len = 1+val, val is 5 bits so length <= 64? tag val is 6 bits (0..63), length 1..64.
			if chunk > 64 {
				chunk = 64
			}
			out = append(out, byte((chunk-1)<<2)|2, byte(off), byte(off>>8))
		}
		n -= chunk
	}
	return out
}

func appendVarint(out []byte, n int) []byte {
	for n >= 128 {
		out = append(out, byte(n)|0x80)
		n >>= 7
	}
	return append(out, byte(n))
}

func readVarint(src []byte) (int, int, error) {
	val := 0
	shift := 0
	for i := 0; i < len(src) && i < 5; i++ {
		b := int(src[i])
		val |= (b & 127) << shift
		if b&128 == 0 {
			return val, i + 1, nil
		}
		shift += 7
	}
	return 0, 0, fmt.Errorf("%w: snappy varint", avutil.ErrInvalidData)
}
