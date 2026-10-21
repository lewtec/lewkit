package avutil

// Bswap16 reverses each pair of bytes. A trailing odd byte is left in place.
func Bswap16(b []byte) {
	if len(b) < 2 {
		return
	}
	if !Scalar() && bswap16Asm(b) {
		return
	}
	for i := 0; i+1 < len(b); i += 2 {
		b[i], b[i+1] = b[i+1], b[i]
	}
}

// Bswap32 reverses each group of four bytes.
func Bswap32(b []byte) {
	if len(b) < 4 {
		return
	}
	if !Scalar() && bswap32Asm(b) {
		return
	}
	for i := 0; i+3 < len(b); i += 4 {
		b[i], b[i+3] = b[i+3], b[i]
		b[i+1], b[i+2] = b[i+2], b[i+1]
	}
}
