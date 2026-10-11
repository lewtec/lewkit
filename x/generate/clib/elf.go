package clib

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	elfPTNull    = 0
	elfPTDynamic = 2
	elfPTInterp  = 3
	elfDTNull    = 0
	elfDTFlags1  = 0x6ffffffb
	elfDF1PIE    = 0x08000000
)

// MakeShared turns an internal-link PIE into a library dlopen accepts.
// glibc refuses a DF_1_PIE object, and a PT_INTERP segment belongs to an
// executable. The file stays ET_DYN.
func MakeShared(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(buf) < 52 || string(buf[:4]) != "\x7fELF" || buf[5] != 1 || buf[6] != 1 || (buf[4] != 1 && buf[4] != 2) {
		return fmt.Errorf("clib: %s is not a little-endian ELF file", path)
	}
	class64 := buf[4] == 2
	var phoff uint64
	var phentsize, phnum uint16
	if class64 {
		if len(buf) < 64 {
			return fmt.Errorf("clib: %s is not an ELF64 file", path)
		}
		phoff = binary.LittleEndian.Uint64(buf[32:])
		phentsize = binary.LittleEndian.Uint16(buf[54:])
		phnum = binary.LittleEndian.Uint16(buf[56:])
		if phentsize < 56 {
			return fmt.Errorf("clib: %s has a short program header", path)
		}
	} else {
		phoff = uint64(binary.LittleEndian.Uint32(buf[28:]))
		phentsize = binary.LittleEndian.Uint16(buf[42:])
		phnum = binary.LittleEndian.Uint16(buf[44:])
		if phentsize < 32 {
			return fmt.Errorf("clib: %s has a short program header", path)
		}
	}
	if phnum == 0 {
		return fmt.Errorf("clib: %s has no program headers", path)
	}
	phend := phoff + uint64(phnum)*uint64(phentsize)
	if phoff > uint64(len(buf)) || phend > uint64(len(buf)) {
		return fmt.Errorf("clib: %s program headers are out of range", path)
	}
	var dynOff, dynSize uint64
	var sawDyn bool
	for i := 0; i < int(phnum); i++ {
		off := int(phoff) + i*int(phentsize)
		pType := binary.LittleEndian.Uint32(buf[off:])
		if pType == elfPTInterp {
			binary.LittleEndian.PutUint32(buf[off:], elfPTNull)
		}
		if pType == elfPTDynamic {
			if class64 {
				dynOff = binary.LittleEndian.Uint64(buf[off+8:])
				dynSize = binary.LittleEndian.Uint64(buf[off+32:])
			} else {
				dynOff = uint64(binary.LittleEndian.Uint32(buf[off+4:]))
				dynSize = uint64(binary.LittleEndian.Uint32(buf[off+16:]))
			}
			sawDyn = true
		}
	}
	if !sawDyn {
		return fmt.Errorf("clib: %s has no dynamic segment", path)
	}
	if dynOff > uint64(len(buf)) || dynSize > uint64(len(buf))-dynOff {
		return fmt.Errorf("clib: %s dynamic segment is out of range", path)
	}
	step := uint64(8)
	if class64 {
		step = 16
	}
	for off := dynOff; off+step <= dynOff+dynSize; off += step {
		var tag, val uint64
		if class64 {
			tag = binary.LittleEndian.Uint64(buf[off:])
			val = binary.LittleEndian.Uint64(buf[off+8:])
		} else {
			tag = uint64(binary.LittleEndian.Uint32(buf[off:]))
			val = uint64(binary.LittleEndian.Uint32(buf[off+4:]))
		}
		if tag == elfDTNull {
			break
		}
		if tag == elfDTFlags1 {
			val &^= elfDF1PIE
			if class64 {
				binary.LittleEndian.PutUint64(buf[off+8:], val)
			} else {
				binary.LittleEndian.PutUint32(buf[off+4:], uint32(val))
			}
		}
	}
	return os.WriteFile(path, buf, info.Mode())
}
