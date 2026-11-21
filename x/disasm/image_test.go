package disasm

import (
	"bytes"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTextUnknown(t *testing.T) {
	_, err := ReadText(bytes.NewReader([]byte("not an object")), 13, "")
	require.ErrorIs(t, err, ErrUnknownFormat)
}

func TestReadTextELF(t *testing.T) {
	raw := ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
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
	raw := ELF64(elf.EM_X86_64, []byte{0x90})
	_, err := ReadText(bytes.NewReader(raw), int64(len(raw)), ".data")
	require.ErrorIs(t, err, ErrNoText)
}

func TestReadTextELFSymbols(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0o644))
	binaryPath := filepath.Join(dir, "out")
	build := exec.Command("go", "build", "-o", binaryPath, source)
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	data, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	got, err := ReadText(bytes.NewReader(data), int64(len(data)), "")
	require.NoError(t, err)
	found := false
	for _, symbol := range got.Symbols {
		if strings.Contains(symbol.Name, "main") {
			found = true
			break
		}
	}
	assert.True(t, found, "symbols: %v", got.Symbols)
}

func TestReadTextELFArm64(t *testing.T) {
	raw := ELF64(elf.EM_AARCH64, []byte{0xff, 0x03, 0xff, 0xb8})
	got, err := ReadText(bytes.NewReader(raw), int64(len(raw)), ".text")
	require.NoError(t, err)
	assert.Equal(t, ArchitectureAArch64, got.Architecture)
	assert.Equal(t, Mode64, got.Mode)
}
