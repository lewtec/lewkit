package jni

import "unicode/utf8"

// encodeMUTF8 appends s as JNI modified UTF-8.
// A NUL byte is the two-byte form, and a rune above U+FFFF is a surrogate pair.
func encodeMUTF8(dst []byte, s string) []byte {
	for _, r := range s {
		switch {
		case r == 0:
			dst = append(dst, 0xC0, 0x80)
		case r < 0x80:
			dst = append(dst, byte(r))
		case r < 0x800:
			dst = append(dst, 0xC0|byte(r>>6), 0x80|byte(r&0x3F))
		case r < 0x10000:
			dst = append(dst, 0xE0|byte(r>>12), 0x80|byte(r>>6&0x3F), 0x80|byte(r&0x3F))
		default:
			r -= 0x10000
			hi := 0xD800 + (r >> 10)
			lo := 0xDC00 + (r & 0x3FF)
			dst = append(dst, mutf8Three(hi)...)
			dst = append(dst, mutf8Three(lo)...)
		}
	}
	return dst
}

func mutf8Three(r rune) []byte {
	return []byte{0xE0 | byte(r>>12), 0x80 | byte(r>>6&0x3F), 0x80 | byte(r&0x3F)}
}

// decodeMUTF8 decodes JNI modified UTF-8. A surrogate pair becomes one rune.
func decodeMUTF8(b []byte) string {
	out := make([]rune, 0, len(b))
	for i := 0; i < len(b); {
		r, size, ok := mutf8Rune(b[i:])
		if !ok {
			out = append(out, utf8.RuneError)
			i++
			continue
		}
		if isHighSurrogate(r) && i+size < len(b) {
			r2, size2, ok2 := mutf8Rune(b[i+size:])
			if ok2 && isLowSurrogate(r2) {
				out = append(out, 0x10000+((r-0xD800)<<10)+(r2-0xDC00))
				i += size + size2
				continue
			}
		}
		out = append(out, r)
		i += size
	}
	return string(out)
}

func mutf8Rune(b []byte) (rune, int, bool) {
	if len(b) == 0 {
		return 0, 0, false
	}
	b0 := b[0]
	switch {
	case b0 < 0x80:
		return rune(b0), 1, true
	case b0&0xE0 == 0xC0 && len(b) >= 2 && b[1]&0xC0 == 0x80:
		return rune(b0&0x1F)<<6 | rune(b[1]&0x3F), 2, true
	case b0&0xF0 == 0xE0 && len(b) >= 3 && b[1]&0xC0 == 0x80 && b[2]&0xC0 == 0x80:
		r := rune(b0&0x0F)<<12 | rune(b[1]&0x3F)<<6 | rune(b[2]&0x3F)
		return r, 3, true
	default:
		return 0, 0, false
	}
}

func isHighSurrogate(r rune) bool { return r >= 0xD800 && r <= 0xDBFF }
func isLowSurrogate(r rune) bool  { return r >= 0xDC00 && r <= 0xDFFF }
