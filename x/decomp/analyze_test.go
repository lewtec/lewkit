package decomp

import (
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/disasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanStrings(t *testing.T) {
	found := scanStrings([]disasm.Section{{
		Address: 0x2000,
		Bytes:   []byte("hello\x00nope"),
	}})
	require.Len(t, found, 1)
	assert.Equal(t, uint64(0x2000), found[0].Address)
	assert.Equal(t, "hello", found[0].Text)
}

func TestOpenReturn(t *testing.T) {
	artifact := openELF(t, []byte{0x90, 0xc3})
	functions := artifact.Functions()
	require.Len(t, functions, 1)
	assert.Equal(t, uint64(0x1000), functions[0].Address)
	assert.Equal(t, "sub_1000", functions[0].Name)
	require.NotEmpty(t, functions[0].Blocks)
	assert.Equal(t, "return", functions[0].Blocks[len(functions[0].Blocks)-1].Kind)
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "nop")
	assert.Contains(t, listing, "ret")
	assert.Contains(t, artifact.Summary(), "functions 1")
}

func TestOpenCallSplitsFunctions(t *testing.T) {
	artifact := openELF(t, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3})
	functions := artifact.Functions()
	require.Len(t, functions, 2)
	assert.Equal(t, uint64(0x1000), functions[0].Address)
	assert.Equal(t, uint64(0x1006), functions[1].Address)
	var sawCall bool
	for _, reference := range functions[0].References {
		if reference.Kind == "call" && reference.To == 0x1006 {
			sawCall = true
		}
	}
	assert.True(t, sawCall, "references: %+v blocks: %+v", functions[0].References, functions[0].Blocks)
}

func TestOpenBranch(t *testing.T) {
	artifact := openELF(t, []byte{0x74, 0x00, 0xc3})
	functions := artifact.Functions()
	require.Len(t, functions, 1)
	require.GreaterOrEqual(t, len(functions[0].Blocks), 2)
	assert.Equal(t, "branch", functions[0].Blocks[0].Kind)
	assert.Equal(t, uint64(0x1002), functions[0].Blocks[0].Target)
	assert.Equal(t, uint64(0x1002), functions[0].Blocks[0].Fall)
}

func TestOpenStringReference(t *testing.T) {
	code := []byte{0x48, 0x8d, 0x3d, 0xf9, 0x0f, 0x00, 0x00, 0xc3}
	raw := elfTextAndRodata(code, append([]byte("hello"), 0))
	project := New()
	artifact, err := project.AddBytes(t.Context(), "a.out", raw)
	require.NoError(t, err)
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Contains(t, function.Strings, "hello")
}

func elfTextAndRodata(code, rodata []byte) []byte {
	const (
		headerSize  = 64
		sectionSize = 64
		sectionNum  = 4
	)
	names := []byte("\x00.text\x00.rodata\x00.shstrtab\x00")
	sectionOff := headerSize
	textOff := sectionOff + sectionNum*sectionSize
	rodataOff := textOff + len(code)
	namesOff := rodataOff + len(rodata)
	buf := make([]byte, namesOff+len(names))
	copy(buf, "\x7fELF")
	buf[4] = 2
	buf[5] = 1
	buf[6] = 1
	binary.LittleEndian.PutUint16(buf[16:], 1)
	binary.LittleEndian.PutUint16(buf[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(buf[20:], 1)
	binary.LittleEndian.PutUint64(buf[24:], 0x1000)
	binary.LittleEndian.PutUint64(buf[40:], uint64(sectionOff))
	binary.LittleEndian.PutUint16(buf[52:], headerSize)
	binary.LittleEndian.PutUint16(buf[58:], sectionSize)
	binary.LittleEndian.PutUint16(buf[60:], sectionNum)
	binary.LittleEndian.PutUint16(buf[62:], 3)
	putSection := func(index, name, kind int, flags, addr, off, size uint64) {
		section := buf[sectionOff+index*sectionSize:]
		binary.LittleEndian.PutUint32(section[0:], uint32(name))
		binary.LittleEndian.PutUint32(section[4:], uint32(kind))
		binary.LittleEndian.PutUint64(section[8:], flags)
		binary.LittleEndian.PutUint64(section[16:], addr)
		binary.LittleEndian.PutUint64(section[24:], off)
		binary.LittleEndian.PutUint64(section[32:], size)
		binary.LittleEndian.PutUint64(section[48:], 1)
	}
	putSection(1, 1, int(elf.SHT_PROGBITS), uint64(elf.SHF_ALLOC|elf.SHF_EXECINSTR), 0x1000, uint64(textOff), uint64(len(code)))
	putSection(2, 7, int(elf.SHT_PROGBITS), uint64(elf.SHF_ALLOC), 0x2000, uint64(rodataOff), uint64(len(rodata)))
	putSection(3, 15, int(elf.SHT_STRTAB), 0, 0, uint64(namesOff), uint64(len(names)))
	copy(buf[textOff:], code)
	copy(buf[rodataOff:], rodata)
	copy(buf[namesOff:], names)
	return buf
}

func TestSlugAndStream(t *testing.T) {
	raw := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
	sum := sha256.Sum256(raw)
	slug := hex.EncodeToString(sum[:])[:16]
	project := New()
	artifact, err := project.AddBytes(t.Context(), "setup.exe", raw)
	require.NoError(t, err)
	assert.Equal(t, slug, artifact.Slug())
	assert.Contains(t, artifact.Summary(), "slug "+slug)
	found, ok := project.Lookup(slug)
	require.True(t, ok)
	assert.Equal(t, artifact.Name(), found.Name())

	require.NoError(t, artifact.SetComment(t.Context(), 0x1000, "entry point"))
	artifact.notes[0x1001] = "on the ret"
	stream := artifact.Stream()
	assert.Contains(t, stream, "; entry point")
	assert.Contains(t, stream, "nop")
	assert.Contains(t, stream, "ret")
	assert.Contains(t, stream, "; note on the ret")
	assert.Less(t, strings.Index(stream, "nop"), strings.Index(stream, "ret"))
}

func TestSetName(t *testing.T) {
	artifact := openELF(t, []byte{0xc3})
	err := artifact.SetName(t.Context(), 0x1000, "start")
	assert.ErrorIs(t, err, ErrNotStored)
	assert.Contains(t, err.Error(), "name was not stored")
	err = artifact.SetNote(t.Context(), 0x1000, "entry")
	assert.ErrorIs(t, err, ErrNotStored)
	assert.Contains(t, err.Error(), "note was not stored")
	err = artifact.SetSignature(t.Context(), 0x1000, "void start(void)")
	assert.ErrorIs(t, err, ErrNotStored)
	assert.Contains(t, err.Error(), "signature was not stored")
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "sub_1000", function.Name)
	assert.Empty(t, function.Signature)
	require.NoError(t, artifact.SetComment(t.Context(), 0x1000, "entry"))
	function, ok = artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "entry", function.Comment)
	err = artifact.SetName(t.Context(), 0x1, "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAddRejectsDuplicateAndEmpty(t *testing.T) {
	project := New()
	raw := disasm.ELF64(elf.EM_X86_64, []byte{0xc3})
	_, err := project.AddBytes(t.Context(), "", raw)
	require.Error(t, err)
	_, err = project.AddBytes(t.Context(), "a.out", raw)
	require.NoError(t, err)
	_, err = project.AddBytes(t.Context(), "a.out", raw)
	assert.ErrorIs(t, err, ErrArtifact)
	assert.Len(t, project.Artifacts(), 1)
}

func openELF(t *testing.T, code []byte) *Artifact {
	t.Helper()
	project := New()
	artifact, err := project.AddBytes(t.Context(), "a.out", disasm.ELF64(elf.EM_X86_64, code))
	require.NoError(t, err)
	return artifact
}
