package disasm

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTextUnknown(t *testing.T) {
	_, err := ReadText(bytes.NewReader([]byte("not an object")), 13, "")
	require.ErrorIs(t, err, ErrUnknownFormat)
}

func TestReadTextELF(t *testing.T) {
	raw := elf64Text(elf.EM_X86_64, []byte{0x90, 0xc3})
	got, err := ReadText(bytes.NewReader(raw), int64(len(raw)), "")
	require.NoError(t, err)
	assert.Equal(t, "elf", got.Format)
	assert.Equal(t, ".text", got.Section)
	assert.Equal(t, ArchitectureX86, got.Architecture)
	assert.Equal(t, Mode64, got.Mode)
	assert.Equal(t, uint64(0x1000), got.Address)
	assert.Equal(t, []byte{0x90, 0xc3}, got.Bytes)
}

func TestReadTextELFNamed(t *testing.T) {
	raw := elf64Text(elf.EM_X86_64, []byte{0x90})
	_, err := ReadText(bytes.NewReader(raw), int64(len(raw)), ".data")
	require.ErrorIs(t, err, ErrNoText)
}

func TestReadTextELFArm64(t *testing.T) {
	raw := elf64Text(elf.EM_AARCH64, []byte{0xff, 0x03, 0xff, 0xb8})
	got, err := ReadText(bytes.NewReader(raw), int64(len(raw)), ".text")
	require.NoError(t, err)
	assert.Equal(t, ArchitectureAArch64, got.Architecture)
	assert.Equal(t, Mode64, got.Mode)
}

func elf64Text(machine elf.Machine, code []byte) []byte {
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
