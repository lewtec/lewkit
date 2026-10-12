package decomp

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/disasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportsAreNamesOnTheThunk(t *testing.T) {
	raw := importExecutable()
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := Open(t.Context(), path)
	require.NoError(t, err)
	defer project.Close()
	artifact, err := project.AddBytes(t.Context(), "imports.exe", raw)
	require.NoError(t, err)

	var names []string
	for _, function := range artifact.Functions() {
		names = append(names, function.Name)
	}
	assert.Contains(t, names, "kernel32!CloseHandle")
	assert.Contains(t, names, "kernel32!GetStdHandle")
	assert.Contains(t, names, "kernel32!ExitProcess")

	thunk, ok := functionNamed(artifact, "kernel32!CloseHandle")
	require.True(t, ok)
	require.Len(t, thunk.Instructions, 1)
	assert.Equal(t, "jmp", thunk.Instructions[0].Mnemonic)
	require.NotEmpty(t, thunk.Blocks)
	assert.Equal(t, uint64(0), thunk.Blocks[0].Target)
	listing, err := artifact.Listing(thunk.Address)
	require.NoError(t, err)
	assert.Contains(t, listing, "jmp dword ptr [0x402000")
	assert.Contains(t, listing, "kernel32!CloseHandle")
	assert.NotContains(t, listing, "->")
	assert.NotContains(t, listing, "GetStdHandle")
	assert.NotContains(t, listing, "ExitProcess")
	assert.Equal(t, 1, strings.Count(listing, "jmp"))

	for _, function := range artifact.Functions() {
		for _, block := range function.Blocks {
			assert.False(t, inSection(artifact, ".idata", block.Target), "block target 0x%x", block.Target)
		}
		for _, reference := range function.References {
			if reference.To == 0x402000 || reference.To == 0x402004 || reference.To == 0x402008 {
				assert.Equal(t, "data", reference.Kind)
			}
		}
	}

	require.NoError(t, artifact.SetNote(t.Context(), 0x402000, "kernel32!CloseHandle"))
	listing, err = artifact.Listing(thunk.Address)
	require.NoError(t, err)
	assert.Contains(t, listing, "; note kernel32!CloseHandle")

	model := &scriptModel{replies: []string{echoCandidate(thunk, thunk.Name, thunk.Name)}}
	artifact.Attach(model)
	result, err := artifact.Match(t.Context(), thunk.Address, MatchConfig{Rounds: 1})
	require.NoError(t, err)
	assert.True(t, result.Matched)
	for _, residual := range result.Residuals {
		assert.NotContains(t, residual, "callee")
	}
}

func TestImportThunkSplitsFromTheCaller(t *testing.T) {
	text := []byte{0x55} // push ebp falls through into the first thunk
	for _, slot := range []uint32{0x402000, 0x402004} {
		text = append(text, 0xff, 0x25)
		var address [4]byte
		binary.LittleEndian.PutUint32(address[:], slot)
		text = append(text, address[:]...)
	}
	data, directory := importSection()
	artifact := openPE(t, buildPE(peProgram{
		imageBase: 0x400000,
		entryRVA:  0x1000,
		sections: []peSection{
			{name: ".text", virtualAddress: 0x1000, virtualSize: uint32(len(text)), data: text, flags: 0x60000020},
			{name: ".idata", virtualAddress: 0x2000, virtualSize: uint32(len(data)), data: data, flags: 0x40000040},
		},
		imports: directory,
	}))

	caller, ok := artifact.Function(0x401000)
	require.True(t, ok)
	assert.Equal(t, "sub_401000", caller.Name)
	for _, instruction := range caller.Instructions {
		assert.NotEqual(t, "jmp", instruction.Mnemonic)
	}
	thunk, ok := artifact.Function(0x401001)
	require.True(t, ok)
	assert.Equal(t, "kernel32!CloseHandle", thunk.Name)
	require.Len(t, thunk.Instructions, 1)
	assert.Equal(t, uint64(6), thunk.Size)
	listing, err := artifact.Listing(0x401001)
	require.NoError(t, err)
	assert.Contains(t, listing, "jmp dword ptr [0x402000")
	assert.Contains(t, listing, "kernel32!CloseHandle")
	assert.NotContains(t, listing, "->")
	assert.NotContains(t, listing, "push")
	assert.NotContains(t, listing, "GetStdHandle")

	direct := []byte{0xe9, 0x01, 0x00, 0x00, 0x00, 0x90}
	direct = append(direct, 0xff, 0x25, 0x00, 0x20, 0x40, 0x00)
	jumped := openPE(t, buildPE(peProgram{
		imageBase: 0x400000,
		entryRVA:  0x1000,
		sections: []peSection{
			{name: ".text", virtualAddress: 0x1000, virtualSize: uint32(len(direct)), data: direct, flags: 0x60000020},
			{name: ".idata", virtualAddress: 0x2000, virtualSize: uint32(len(data)), data: data, flags: 0x40000040},
		},
		imports: directory,
	}))
	thunk, ok = jumped.Function(0x401006)
	require.True(t, ok)
	assert.Equal(t, "kernel32!CloseHandle", thunk.Name)
	require.Len(t, thunk.Instructions, 1)
	caller, ok = jumped.Function(0x401000)
	require.True(t, ok)
	for _, instruction := range caller.Instructions {
		assert.NotEqual(t, uint64(0x401006), instruction.Address)
	}
}

func TestExportsAreFunctionStarts(t *testing.T) {
	artifact := openPE(t, exportExecutable())
	function, ok := artifact.Function(0x401000)
	require.True(t, ok)
	assert.Equal(t, "ISArcExtract", function.Name)
	listing, err := artifact.Listing(0x401000)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(listing, "ISArcExtract:\n"), listing)
	err = artifact.SetName(t.Context(), 0x401001, "inside")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "function not found")
}

func TestFunctionStopsAtTheCode(t *testing.T) {
	small := openPE(t, textExecutable(".text", []byte{0xc3, 0x55}, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}))
	entry, err := small.Listing(small.Entry())
	require.NoError(t, err)
	assert.Contains(t, entry, "ret")
	assert.NotContains(t, entry, "add byte ptr [eax], al")
	assert.NotContains(t, entry, "push")
	other, ok := small.Function(small.Entry() + 1)
	require.True(t, ok)
	assert.NotEqual(t, small.Entry(), other.Address)
	otherListing, err := small.Listing(other.Address)
	require.NoError(t, err)
	assert.Contains(t, otherListing, "push")

	body := bytes.Repeat([]byte{0x90}, 0xbe8)
	body[0] = 0xc3
	padding := append(bytes.Repeat([]byte{0x00}, 16), 0x69, 0x00, 0x6f, 0x00, 0x6e, 0x00)
	wide := openPE(t, textExecutable(".itext", body, padding))
	assert.Contains(t, wide.Summary(), ".itext")
	assert.Contains(t, wide.Summary(), "0xbe8")
	entry, err = wide.Listing(wide.Entry())
	require.NoError(t, err)
	assert.NotContains(t, entry, "imul")
	assert.NotContains(t, entry, "0x6e006f")
	assert.NotContains(t, entry, "add byte ptr [eax], al")
	for _, function := range wide.Functions() {
		for _, instruction := range function.Instructions {
			assert.NotEqual(t, "add", instruction.Mnemonic)
			assert.NotContains(t, instruction.Operands, "0x6e006f")
		}
	}
}

func TestUTF16AndASCIIStrings(t *testing.T) {
	rodata := make([]byte, 48)
	copy(rodata, utf16Text("Setup"))
	copy(rodata[16:], append([]byte("arc.ini"), 0))
	copy(rodata[32:], append([]byte("System"), 0))
	code := []byte{
		0xb8, 0x00, 0x20, 0x00, 0x00,
		0xb8, 0x10, 0x20, 0x00, 0x00,
		0xc3,
	}
	raw := elfTextAndRodata(code, rodata)
	project := New()
	artifact, err := project.AddBytes(t.Context(), "strings.elf", raw)
	require.NoError(t, err)
	var texts []string
	for _, item := range artifact.Strings() {
		texts = append(texts, item.Text)
		assert.NotZero(t, item.Address)
	}
	assert.Contains(t, texts, "Setup")
	assert.Contains(t, texts, "arc.ini")
	assert.Contains(t, texts, "System")
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, `"Setup"`)
	assert.Contains(t, listing, `"arc.ini"`)

	stored := filepath.Join(t.TempDir(), "program.decomp")
	file, err := Open(t.Context(), stored)
	require.NoError(t, err)
	kept, err := file.AddBytes(t.Context(), "strings.elf", raw)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	file, err = Open(t.Context(), stored)
	require.NoError(t, err)
	defer file.Close()
	kept, ok := file.Artifact("strings.elf")
	require.True(t, ok)
	texts = nil
	for _, item := range kept.Strings() {
		texts = append(texts, item.Text)
	}
	assert.Contains(t, texts, "Setup")
	assert.Contains(t, texts, "arc.ini")
	_ = kept
}

func TestCallOperandUsesTheFunctionName(t *testing.T) {
	raw := disasm.ELF64(elf.EM_X86_64, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3})
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := Open(t.Context(), path)
	require.NoError(t, err)
	artifact, err := project.AddBytes(t.Context(), "call.elf", raw)
	require.NoError(t, err)
	require.NoError(t, artifact.SetName(t.Context(), 0x1006, "StringAssign"))
	require.NoError(t, project.Close())

	project, err = Open(t.Context(), path)
	require.NoError(t, err)
	defer project.Close()
	artifact, ok := project.Artifact("call.elf")
	require.True(t, ok)
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "call 0x1006 <StringAssign>")
	refs := artifact.XRefs(0x1006)
	require.NotEmpty(t, refs)
	assert.Equal(t, uint64(0x1000), refs[0].From)
	assert.Equal(t, "call", refs[0].Kind)
}

func TestUnreachableInstructionIsItsOwnFunction(t *testing.T) {
	artifact := openELF(t, []byte{0xc3, 0x55})
	functions := artifact.Functions()
	require.Len(t, functions, 2)
	assert.Equal(t, uint64(0x1000), functions[0].Address)
	assert.Equal(t, uint64(0x1001), functions[1].Address)
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.NotContains(t, listing, "push")
}

func TestExplainKeepsSourceWhenMatchDiffers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := Open(t.Context(), path)
	require.NoError(t, err)
	artifact, err := project.AddBytes(t.Context(), "a.out", disasm.ELF64(elf.EM_X86_64, []byte{0xc3}))
	require.NoError(t, err)
	artifact.Attach(&scriptModel{replies: []string{`{"source":"void entry(void) { return; }"}`}})
	result, err := artifact.Explain(t.Context(), 0x1000, MatchConfig{Rounds: 1})
	require.NoError(t, err)
	assert.Equal(t, "void entry(void) { return; }", result.Source)
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "void entry(void) { return; }")

	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	artifact.Attach(&scriptModel{replies: []string{`{"name":"nope","signature":"void nope(void)","source":"void nope(void) { return; }","blocks":[]}`}})
	_, err = artifact.Match(t.Context(), 0x1000, MatchConfig{Rounds: 1})
	assert.ErrorIs(t, err, ErrUnmatched)
	function, ok = artifact.Function(0x1000)
	require.True(t, ok)
	assert.Contains(t, function.Source, "void entry(void) { return; }")
	require.NoError(t, artifact.SetSignature(t.Context(), 0x1000, "void entry(void)"))
	require.NoError(t, project.Close())

	project, err = Open(t.Context(), path)
	require.NoError(t, err)
	defer project.Close()
	artifact, ok = project.Artifact("a.out")
	require.True(t, ok)
	listing, err = artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "signature void entry(void)")
	assert.Contains(t, listing, "void entry(void) { return; }")
}

func TestProjectRemembersTheFile(t *testing.T) {
	if os.Getenv("DECOMP_RELOAD") == "1" {
		project, err := Open(t.Context(), os.Getenv("DECOMP_PROJECT"))
		require.NoError(t, err)
		defer project.Close()
		for _, image := range strings.Split(os.Getenv("DECOMP_IMAGES"), "\n") {
			data, err := os.ReadFile(image)
			require.NoError(t, err)
			artifact, err := project.Remember(t.Context(), image, data)
			require.NoError(t, err)
			t.Log(artifact.Summary())
			_, _ = os.Stdout.WriteString(artifact.Summary())
		}
		return
	}

	root := t.TempDir()
	firstPath := filepath.Join(root, "one", "setup.exe")
	secondPath := filepath.Join(root, "two", "setup.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(firstPath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(secondPath), 0o755))
	first := disasm.ELF64(elf.EM_X86_64, []byte{0xc3})
	second := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
	require.NoError(t, os.WriteFile(firstPath, first, 0o644))
	require.NoError(t, os.WriteFile(secondPath, second, 0o644))

	database := filepath.Join(root, "program.decomp")
	project, err := Open(t.Context(), database)
	require.NoError(t, err)
	loaded, err := project.Remember(t.Context(), firstPath, first)
	require.NoError(t, err)
	assert.Contains(t, loaded.Summary(), ".text 0x1")
	other, err := project.Remember(t.Context(), secondPath, second)
	require.NoError(t, err)
	assert.Contains(t, other.Summary(), ".text 0x2")
	assert.NotEqual(t, loaded.Name(), other.Name())

	changed := disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0x90, 0xc3})
	require.NoError(t, os.WriteFile(firstPath, changed, 0o644))
	_, err = project.Remember(t.Context(), firstPath, changed)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImage)
	assert.Contains(t, err.Error(), "stored image")
	assert.Contains(t, err.Error(), "new bytes")
	assert.Contains(t, err.Error(), imageDigest(first))
	assert.Contains(t, err.Error(), imageDigest(changed))
	require.NoError(t, os.WriteFile(firstPath, first, 0o644))
	require.NoError(t, project.Close())

	command := exec.Command(os.Args[0], "-test.run", "^TestProjectRemembersTheFile$")
	command.Env = append(os.Environ(),
		"DECOMP_RELOAD=1",
		"DECOMP_PROJECT="+database,
		"DECOMP_IMAGES="+firstPath+"\n"+secondPath,
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	text := string(output)
	assert.Contains(t, text, ".text 0x1")
	assert.Contains(t, text, ".text 0x2")
	assert.NotContains(t, strings.ToLower(text), "capstone")
}

func TestSameBytesUseOneSlug(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "one", "setup.exe")
	secondPath := filepath.Join(root, "two", "setup.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(firstPath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(secondPath), 0o755))
	raw := disasm.ELF64(elf.EM_X86_64, []byte{0xc3})
	require.NoError(t, os.WriteFile(firstPath, raw, 0o644))
	require.NoError(t, os.WriteFile(secondPath, raw, 0o644))
	project, err := Open(t.Context(), filepath.Join(root, "program.decomp"))
	require.NoError(t, err)
	defer project.Close()
	first, err := project.Remember(t.Context(), firstPath, raw)
	require.NoError(t, err)
	second, err := project.Remember(t.Context(), secondPath, raw)
	require.NoError(t, err)
	assert.Equal(t, first.Slug(), second.Slug())
	assert.Len(t, project.Artifacts(), 1)
}

func openPE(t *testing.T, raw []byte) *Artifact {
	t.Helper()
	project := New()
	artifact, err := project.AddBytes(t.Context(), "sample.exe", raw)
	require.NoError(t, err)
	return artifact
}

func functionNamed(artifact *Artifact, name string) (Function, bool) {
	for _, function := range artifact.Functions() {
		if function.Name == name {
			return function, true
		}
	}
	return Function{}, false
}

func inSection(artifact *Artifact, name string, address uint64) bool {
	if address == 0 {
		return false
	}
	for _, section := range artifact.sections {
		if section.Name != name {
			continue
		}
		if address >= section.Address && address < section.Address+section.Size {
			return true
		}
	}
	return false
}

func utf16Text(text string) []byte {
	out := make([]byte, 0, len(text)*2+2)
	for index := 0; index < len(text); index++ {
		out = append(out, text[index], 0)
	}
	return append(out, 0, 0)
}

func importExecutable() []byte {
	text := make([]byte, 0, 24)
	for _, slot := range []uint32{0x402000, 0x402004, 0x402008} {
		text = append(text, 0xff, 0x25)
		var address [4]byte
		binary.LittleEndian.PutUint32(address[:], slot)
		text = append(text, address[:]...)
		text = append(text, 0x90, 0x90)
	}
	data, directory := importSection()
	return buildPE(peProgram{
		imageBase: 0x400000,
		entryRVA:  0x1000,
		sections: []peSection{
			{name: ".text", virtualAddress: 0x1000, virtualSize: uint32(len(text)), data: text, flags: 0x60000020},
			{name: ".idata", virtualAddress: 0x2000, virtualSize: uint32(len(data)), data: data, flags: 0x40000040},
		},
		imports: directory,
	})
}

func importSection() ([]byte, peDirectory) {
	var buf []byte
	buf = append(buf, make([]byte, 16)...)
	buf = append(buf, make([]byte, 16)...)
	buf = append(buf, make([]byte, 40)...)
	dllAt := len(buf)
	buf = append(buf, []byte("kernel32.dll\x00")...)
	names := []string{"CloseHandle", "GetStdHandle", "ExitProcess"}
	nameAt := make([]uint32, len(names))
	for index, name := range names {
		nameAt[index] = 0x2000 + uint32(len(buf))
		buf = append(buf, 0, 0)
		buf = append(buf, name...)
		buf = append(buf, 0)
	}
	for index, rva := range nameAt {
		binary.LittleEndian.PutUint32(buf[index*4:], rva)
		binary.LittleEndian.PutUint32(buf[16+index*4:], rva)
	}
	binary.LittleEndian.PutUint32(buf[32:], 0x2010)
	binary.LittleEndian.PutUint32(buf[32+12:], 0x2000+uint32(dllAt))
	binary.LittleEndian.PutUint32(buf[32+16:], 0x2000)
	return buf, peDirectory{virtualAddress: 0x2020, size: 40}
}

func exportExecutable() []byte {
	text := []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc3}
	data := make([]byte, 40)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], 1)
	binary.LittleEndian.PutUint32(data[28:], 0x2028)
	binary.LittleEndian.PutUint32(data[32:], 0x202c)
	binary.LittleEndian.PutUint32(data[36:], 0x2030)
	function := make([]byte, 4)
	binary.LittleEndian.PutUint32(function, 0x1000)
	name := make([]byte, 4)
	binary.LittleEndian.PutUint32(name, 0x2032)
	data = append(data, function...)
	data = append(data, name...)
	data = append(data, 0, 0)
	data = append(data, []byte("ISArcExtract\x00")...)
	return buildPE(peProgram{
		imageBase: 0x400000,
		entryRVA:  0x1000,
		sections: []peSection{
			{name: ".text", virtualAddress: 0x1000, virtualSize: uint32(len(text)), data: text, flags: 0x60000020},
			{name: ".rdata", virtualAddress: 0x2000, virtualSize: uint32(len(data)), data: data, flags: 0x40000040},
		},
		exports: peDirectory{virtualAddress: 0x2000, size: 40},
	})
}

func textExecutable(name string, virtual, padding []byte) []byte {
	data := append(append([]byte{}, virtual...), padding...)
	return buildPE(peProgram{
		imageBase: 0x400000,
		entryRVA:  0x1000,
		sections: []peSection{{
			name:           name,
			virtualAddress: 0x1000,
			virtualSize:    uint32(len(virtual)),
			data:           data,
			flags:          0x60000020,
		}},
	})
}

type peDirectory struct {
	virtualAddress uint32
	size           uint32
}

type peSection struct {
	name           string
	virtualAddress uint32
	virtualSize    uint32
	data           []byte
	flags          uint32
}

type peProgram struct {
	imageBase uint32
	entryRVA  uint32
	sections  []peSection
	imports   peDirectory
	exports   peDirectory
}

func buildPE(program peProgram) []byte {
	const (
		fileAlign    = 0x200
		sectionAlign = 0x1000
		peOffset     = 0x80
		optSize      = 224
	)
	headerEnd := uint32(peOffset + 4 + 20 + optSize + 40*len(program.sections))
	rawAt := uint32(fileAlign)
	if headerEnd > rawAt {
		rawAt = (headerEnd + fileAlign - 1) &^ (fileAlign - 1)
	}
	type laid struct {
		section peSection
		raw     uint32
		rawSize uint32
	}
	laidOut := make([]laid, len(program.sections))
	cursor := rawAt
	var imageEnd uint32
	for index, section := range program.sections {
		rawSize := uint32(len(section.data))
		if rawSize == 0 {
			rawSize = fileAlign
		}
		padded := (rawSize + fileAlign - 1) &^ (fileAlign - 1)
		laidOut[index] = laid{section: section, raw: cursor, rawSize: padded}
		cursor += padded
		virtual := section.virtualSize
		if virtual == 0 {
			virtual = uint32(len(section.data))
		}
		end := section.virtualAddress + virtual
		aligned := (end + sectionAlign - 1) &^ (sectionAlign - 1)
		if aligned > imageEnd {
			imageEnd = aligned
		}
	}
	file := make([]byte, cursor)
	file[0] = 'M'
	file[1] = 'Z'
	binary.LittleEndian.PutUint32(file[0x3c:], peOffset)
	file[peOffset] = 'P'
	file[peOffset+1] = 'E'
	coff := peOffset + 4
	binary.LittleEndian.PutUint16(file[coff:], 0x14c)
	binary.LittleEndian.PutUint16(file[coff+2:], uint16(len(program.sections)))
	binary.LittleEndian.PutUint16(file[coff+16:], optSize)
	binary.LittleEndian.PutUint16(file[coff+18:], 0x0102)
	optional := coff + 20
	binary.LittleEndian.PutUint16(file[optional:], 0x10b)
	binary.LittleEndian.PutUint32(file[optional+16:], program.entryRVA)
	binary.LittleEndian.PutUint32(file[optional+28:], program.imageBase)
	binary.LittleEndian.PutUint32(file[optional+32:], sectionAlign)
	binary.LittleEndian.PutUint32(file[optional+36:], fileAlign)
	binary.LittleEndian.PutUint16(file[optional+48:], 4)
	binary.LittleEndian.PutUint32(file[optional+56:], imageEnd)
	binary.LittleEndian.PutUint32(file[optional+60:], uint32(rawAt))
	binary.LittleEndian.PutUint16(file[optional+68:], 3)
	binary.LittleEndian.PutUint32(file[optional+92:], 16)
	directories := optional + 96
	binary.LittleEndian.PutUint32(file[directories:], program.exports.virtualAddress)
	binary.LittleEndian.PutUint32(file[directories+4:], program.exports.size)
	binary.LittleEndian.PutUint32(file[directories+8:], program.imports.virtualAddress)
	binary.LittleEndian.PutUint32(file[directories+12:], program.imports.size)
	sectionHeader := optional + optSize
	for index, item := range laidOut {
		header := file[sectionHeader+index*40:]
		copy(header[:8], item.section.name)
		binary.LittleEndian.PutUint32(header[8:], item.section.virtualSize)
		binary.LittleEndian.PutUint32(header[12:], item.section.virtualAddress)
		binary.LittleEndian.PutUint32(header[16:], item.rawSize)
		binary.LittleEndian.PutUint32(header[20:], item.raw)
		binary.LittleEndian.PutUint32(header[36:], item.section.flags)
		copy(file[item.raw:], item.section.data)
	}
	return file
}
