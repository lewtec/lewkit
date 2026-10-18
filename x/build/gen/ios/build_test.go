package ios

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gotool.TestExit(m)
}

func TestBuildAssemblesGoApp(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := filepath.Join(t.TempDir(), "Demo.app")
	result, err := Build(t.Context(), BuildOptions{
		Config: Config{
			PackageID:   "br.tec.lew.demo",
			AppName:     "Demo",
			VersionName: "1.2.3",
			VersionCode: 5,
			GoMain:      mainDir,
		},
		BaseDir: mainDir,
		WorkDir: t.TempDir(),
		OutApp:  out,
		GOARCH:  "arm64",
	})
	require.NoError(t, err)
	require.Equal(t, out, result.AppPath)
	exe := filepath.Join(out, "Demo")
	require.Equal(t, uint32(platformIOS), machoBuildPlatform(t, exe))
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	require.Contains(t, string(plist), "<key>CFBundleExecutable</key>\n\t<string>Demo</string>")
	require.Contains(t, string(plist), "<string>iPhoneOS</string>")
	require.Contains(t, string(plist), "<key>MinimumOSVersion</key>\n\t<string>13.0</string>")
	_, err = os.Stat(filepath.Join(out, "Icon@3x.png"))
	require.NoError(t, err)
}

func machoBuildPlatform(t *testing.T, path string) uint32 {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Greater(t, len(raw), macho64Header)
	ncmds := binary.LittleEndian.Uint32(raw[16:20])
	off := macho64Header
	for range ncmds {
		require.Less(t, off+8, len(raw))
		cmd := binary.LittleEndian.Uint32(raw[off:])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		require.GreaterOrEqual(t, size, 8)
		if cmd == loadCmdBuildVersion && off+buildVersionSize <= len(raw) {
			return binary.LittleEndian.Uint32(raw[off+8:])
		}
		off += size
	}
	require.FailNow(t, "missing LC_BUILD_VERSION")
	return 0
}
