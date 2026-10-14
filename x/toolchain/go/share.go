package gotool

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

// MakeShared turns an internal-link PIE into a file a dynamic loader can
// open. glibc rejects DF_1_PIE, and PT_INTERP marks an executable. The
// image stays ET_DYN. The linker will not make this with -buildmode=c-shared
// unless linkmode=external, and that mode needs cgo.
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
		return fmt.Errorf("%s %w", path, ErrIsNotALittleEndianElf64File)
	}
	phoff := binary.LittleEndian.Uint64(buf[32:])
	phentsize := binary.LittleEndian.Uint16(buf[54:])
	phnum := binary.LittleEndian.Uint16(buf[56:])
	if phentsize < 56 || phnum == 0 {
		return fmt.Errorf("%s %w", path, ErrHasABadProgramHeaderTable)
	}
	phend := phoff + uint64(phnum)*uint64(phentsize)
	if phoff > uint64(len(buf)) || phend > uint64(len(buf)) {
		return fmt.Errorf("%s %w", path, ErrProgramHeadersAreOutOfRange)
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
		return fmt.Errorf("%s %w", path, ErrHasNoDynamicSegment)
	}
	if dynOff > uint64(len(buf)) || dynSize > uint64(len(buf))-dynOff {
		return fmt.Errorf("%s %w", path, ErrDynamicSegmentIsOutOfRange)
	}
	const step = 16
	for off := dynOff; off+step <= dynOff+dynSize; off += step {
		tag := binary.LittleEndian.Uint64(buf[off:])
		if tag == elfDTNull {
			break
		}
		if tag == elfDTFlags1 {
			val := binary.LittleEndian.Uint64(buf[off+8:])
			val &^= elfDF1PIE
			binary.LittleEndian.PutUint64(buf[off+8:], val)
		}
	}
	return os.WriteFile(path, buf, info.Mode())
}
