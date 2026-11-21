package disasm

import (
	"debug/elf"
	"encoding/binary"
)

// ELF64 is a minimal ELF64 object with one executable .text section.
func ELF64(machine elf.Machine, code []byte) []byte {
	const (
		ehsize  = 64
		shsize  = 64
		shnum   = 3
		shoff   = ehsize
		textoff = ehsize + shnum*shsize
	)
	shstr := []byte("\x00.text\x00.shstrtab\x00")
	buf := make([]byte, textoff+len(code)+len(shstr))
	copy(buf[0:], "\x7fELF")
	buf[4] = 2
	buf[5] = 1
	buf[6] = 1
	binary.LittleEndian.PutUint16(buf[16:], 1)
	binary.LittleEndian.PutUint16(buf[18:], uint16(machine))
	binary.LittleEndian.PutUint32(buf[20:], 1)
	binary.LittleEndian.PutUint64(buf[40:], shoff)
	binary.LittleEndian.PutUint16(buf[52:], ehsize)
	binary.LittleEndian.PutUint16(buf[58:], shsize)
	binary.LittleEndian.PutUint16(buf[60:], shnum)
	binary.LittleEndian.PutUint16(buf[62:], 2)

	text := buf[shoff+shsize:]
	binary.LittleEndian.PutUint32(text[0:], 1)
	binary.LittleEndian.PutUint32(text[4:], 1)
	binary.LittleEndian.PutUint64(text[8:], uint64(elf.SHF_ALLOC|elf.SHF_EXECINSTR))
	binary.LittleEndian.PutUint64(text[16:], 0x1000)
	binary.LittleEndian.PutUint64(text[24:], uint64(textoff))
	binary.LittleEndian.PutUint64(text[32:], uint64(len(code)))
	binary.LittleEndian.PutUint64(text[48:], 1)

	strtab := buf[shoff+2*shsize:]
	binary.LittleEndian.PutUint32(strtab[0:], 7)
	binary.LittleEndian.PutUint32(strtab[4:], 3)
	binary.LittleEndian.PutUint64(strtab[24:], uint64(textoff+len(code)))
	binary.LittleEndian.PutUint64(strtab[32:], uint64(len(shstr)))
	binary.LittleEndian.PutUint64(strtab[48:], 1)

	copy(buf[textoff:], code)
	copy(buf[textoff+len(code):], shstr)
	return buf
}
