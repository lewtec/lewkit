package clib

import "unsafe"

// Store writes the low bytes of word into dst at off.
// An aligned eight-byte write is one store, so a pointer field is not torn.
func Store(dst unsafe.Pointer, size, off int, word uint64) {
	if dst == nil || size <= 0 || off < 0 || off >= size {
		return
	}
	n := size - off
	if n > 8 {
		n = 8
	}
	if n == 8 && off%8 == 0 {
		*(*uint64)(unsafe.Add(dst, off)) = word
		return
	}
	raw := unsafe.Slice((*byte)(dst), size)
	for i := 0; i < n; i++ {
		raw[off+i] = byte(word)
		word >>= 8
	}
}

// Load reads up to eight little-endian bytes from src at off.
func Load(src unsafe.Pointer, size, off int) uint64 {
	if src == nil || size <= 0 || off < 0 || off >= size {
		return 0
	}
	n := size - off
	if n > 8 {
		n = 8
	}
	if n == 8 && off%8 == 0 {
		return *(*uint64)(unsafe.Add(src, off))
	}
	raw := unsafe.Slice((*byte)(src), size)
	var out uint64
	for i := 0; i < n; i++ {
		out |= uint64(raw[off+i]) << (uint(i) * 8)
	}
	return out
}

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
