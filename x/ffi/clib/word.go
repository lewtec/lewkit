package clib

// U32 reads four little-endian bytes from the argument image at off.
// The 386 cdecl entry packs the C stack into Args. A short or a byte
// still occupies four bytes there, and the value is in the low bytes.
func U32(args *[8]uint64, off int) uint32 {
	if args == nil || off < 0 || off > 60 {
		return 0
	}
	var raw [64]byte
	for i, v := range args {
		for j := 0; j < 8; j++ {
			raw[i*8+j] = byte(v)
			v >>= 8
		}
	}
	return uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
}

// U64 reads eight little-endian bytes from the argument image at off.
func U64(args *[8]uint64, off int) uint64 {
	if args == nil || off < 0 || off > 56 {
		return 0
	}
	return uint64(U32(args, off)) | uint64(U32(args, off+4))<<32
}
