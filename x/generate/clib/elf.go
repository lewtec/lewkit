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
	if len(buf) < 64 || string(buf[:4]) != "\x7fELF" || buf[4] != 2 || buf[5] != 1 || buf[6] != 1 {
		return fmt.Errorf("clib: %s is not an ELF64 little-endian file", path)
	}
	phoff := binary.LittleEndian.Uint64(buf[32:])
	phentsize := binary.LittleEndian.Uint16(buf[54:])
	phnum := binary.LittleEndian.Uint16(buf[56:])
	if phentsize < 56 || phnum == 0 {
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
			dynOff = binary.LittleEndian.Uint64(buf[off+8:])
			dynSize = binary.LittleEndian.Uint64(buf[off+32:])
			sawDyn = true
		}
	}
	if !sawDyn {
		return fmt.Errorf("clib: %s has no dynamic segment", path)
	}
	if dynOff > uint64(len(buf)) || dynSize > uint64(len(buf))-dynOff {
		return fmt.Errorf("clib: %s dynamic segment is out of range", path)
	}
	for off := dynOff; off+16 <= dynOff+dynSize; off += 16 {
		tag := binary.LittleEndian.Uint64(buf[off:])
		if tag == elfDTNull {
			break
		}
		if tag == elfDTFlags1 {
			val := binary.LittleEndian.Uint64(buf[off+8:])
			binary.LittleEndian.PutUint64(buf[off+8:], val&^elfDF1PIE)
		}
	}
	return os.WriteFile(path, buf, info.Mode())
}
