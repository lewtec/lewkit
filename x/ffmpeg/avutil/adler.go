package avutil

// Adler32 is av_adler32_update(0, data). The seed is zero, so the first
// sums start at zero rather than the zlib seed of one.
func Adler32(data []byte) uint32 {
	var s1, s2 uint64
	for _, b := range data {
		s1 += uint64(b)
		s2 += s1
	}
	s1 %= 65521
	s2 %= 65521
	return uint32(s2<<16 | s1)
}
