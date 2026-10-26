package ios

import (
	"archive/zip"
	"encoding/binary"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/lewkit/x/build/gocmd/lewkit"
	"github.com/lewtec/lewkit/x/build/sign"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gotool.TestExit(m)
}

func TestIOSLaunchScreenNamesSplash(t *testing.T) {
	plist := iosPlist(Config{
		PackageID:   "br.tec.lew.demo",
		AppName:     "Demo",
		VersionName: "1.2.3",
		VersionCode: 1,
	}, "arm64", true)
	require.Contains(t, plist, "<key>UILaunchScreen</key>")
	require.Contains(t, plist, "<key>UIImageName</key>\n\t\t<string>SplashLogo</string>")
	require.Contains(t, plist, "<key>UIImageRespectsSafeAreaInsets</key>\n\t\t<true/>")
}

func TestWriteBundleIconsWritesSplash(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "source"), 0o755))
	mark := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	file, err := os.Create(filepath.Join(root, "source", "master.png"))
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, mark))
	require.NoError(t, file.Close())
	app := t.TempDir()
	require.NoError(t, writeBundleIcons(root, app))
	for _, name := range []string{"Icon@2x.png", "Icon@3x.png", "SplashLogo.png", "SplashLogo@2x.png", "SplashLogo@3x.png"} {
		info, err := os.Stat(filepath.Join(app, name))
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0))
	}
	logo, err := os.Open(filepath.Join(app, "SplashLogo.png"))
	require.NoError(t, err)
	defer logo.Close()
	cfg, err := png.DecodeConfig(logo)
	require.NoError(t, err)
	require.Equal(t, SplashPointSize, cfg.Width)
	require.Equal(t, SplashPointSize, cfg.Height)
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
	require.Contains(t, string(plist), "<key>UIRequiredDeviceCapabilities</key>")
	requireIOSFrameworks(t, exe)
	ipa := filepath.Join(t.TempDir(), "Demo.ipa")
	require.NoError(t, Pack(t.Context(), out, ipa))
	reader, err := zip.OpenReader(ipa)
	require.NoError(t, err)
	defer reader.Close()
	member, err := reader.Open("Payload/Demo.app/Demo")
	require.NoError(t, err)
	defer member.Close()
	magic := make([]byte, 4)
	_, err = io.ReadFull(member, magic)
	require.NoError(t, err)
	require.Equal(t, []byte{0xCF, 0xFA, 0xED, 0xFE}, magic)
}

func TestBuildAssemblesSimulator(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := filepath.Join(t.TempDir(), "Demo.app")
	_, err := Build(t.Context(), BuildOptions{
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
		SDK:     string(SDKSimulator),
	})
	require.NoError(t, err)
	exe := filepath.Join(out, "Demo")
	require.Equal(t, uint32(platformIOSSim), machoBuildPlatform(t, exe))
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	require.Contains(t, string(plist), "<string>iPhoneSimulator</string>")
	require.NotContains(t, string(plist), "UIRequiredDeviceCapabilities")
	requireIOSFrameworks(t, exe)
	require.NoError(t, sign.SignTree(nil, out, "br.tec.lew.demo"))
	require.Equal(t, uint32(platformIOSSim), machoBuildPlatform(t, exe))
	requireIOSFrameworks(t, exe)
}

func requireIOSFrameworks(t *testing.T, exe string) {
	t.Helper()
	names := machoDylibNames(t, exe)
	require.Contains(t, names, "/usr/lib/libSystem.B.dylib")
	require.Contains(t, names, "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
	for _, name := range names {
		require.NotContains(t, name, "/Versions/A/")
	}
	// _rt0_arm64_ios is the instruction word 0. dyld executes that entry.
	require.NotEqual(t, uint32(0), machoEntryInsn(t, exe))
}

func machoEntryInsn(t *testing.T, path string) uint32 {
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
		require.LessOrEqual(t, off+size, len(raw))
		if cmd == loadCmdMain && off+16 <= len(raw) {
			entry := binary.LittleEndian.Uint64(raw[off+8:])
			require.LessOrEqual(t, entry+4, uint64(len(raw)))
			return binary.LittleEndian.Uint32(raw[entry:])
		}
		off += size
	}
	require.FailNow(t, "missing LC_MAIN")
	return 0
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
