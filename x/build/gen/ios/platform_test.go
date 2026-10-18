package ios

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteIOSFrameworks(t *testing.T) {
	cf := "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
	sec := "/System/Library/Frameworks/Security.framework/Versions/A/Security"
	lib := "/usr/lib/libSystem.B.dylib"
	flat := "/System/Library/Frameworks/UIKit.framework/UIKit"
	path := filepath.Join(t.TempDir(), "exe")
	require.NoError(t, os.WriteFile(path, fakeDylibMachO(cf, sec, lib, flat), 0o644))
	require.NoError(t, rewriteIOSFrameworks(path))
	got := machoDylibNames(t, path)
	require.Equal(t, []string{
		"/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
		"/System/Library/Frameworks/Security.framework/Security",
		lib,
		flat,
	}, got)
	require.NoError(t, rewriteIOSFrameworks(path))
	require.Equal(t, got, machoDylibNames(t, path))
}

func TestRewriteIOSFrameworksRejectsNonMachO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exe")
	require.NoError(t, os.WriteFile(path, []byte("not a macho"), 0o644))
	require.ErrorIs(t, rewriteIOSFrameworks(path), ErrIOSFramework)
}

func fakeDylibMachO(names ...string) []byte {
	var cmds []byte
	for _, name := range names {
		cmds = append(cmds, dylibCmd(name)...)
	}
	hdr := make([]byte, macho64Header)
	binary.LittleEndian.PutUint32(hdr[0:], 0xFEEDFACF)
	binary.LittleEndian.PutUint32(hdr[16:], uint32(len(names)))
	binary.LittleEndian.PutUint32(hdr[20:], uint32(len(cmds)))
	return append(hdr, cmds...)
}

func dylibCmd(name string) []byte {
	raw := dylibNameOffset + len(name) + 1
	size := (raw + 7) &^ 7
	cmd := make([]byte, size)
	binary.LittleEndian.PutUint32(cmd[0:], loadCmdLoadDylib)
	binary.LittleEndian.PutUint32(cmd[4:], uint32(size))
	binary.LittleEndian.PutUint32(cmd[8:], dylibNameOffset)
	copy(cmd[dylibNameOffset:], name)
	return cmd
}

func machoDylibNames(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Greater(t, len(raw), macho64Header)
	ncmds := binary.LittleEndian.Uint32(raw[16:20])
	off := macho64Header
	var names []string
	for range ncmds {
		require.Less(t, off+8, len(raw))
		cmd := binary.LittleEndian.Uint32(raw[off:])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		require.GreaterOrEqual(t, size, 8)
		require.LessOrEqual(t, off+size, len(raw))
		if isDylibCommand(cmd) {
			nameOff := int(binary.LittleEndian.Uint32(raw[off+8:]))
			require.GreaterOrEqual(t, nameOff, dylibNameOffset)
			text, ok := cstring(raw[off+nameOff : off+size])
			require.True(t, ok)
			names = append(names, text)
		}
		off += size
	}
	return names
}
