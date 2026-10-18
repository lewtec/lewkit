package mac

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	exe, err := os.ReadFile(filepath.Join(out, "Contents", "MacOS", "Demo"))
	require.NoError(t, err)
	require.Equal(t, []byte{0xcf, 0xfa, 0xed, 0xfe}, exe[:4])
	_, err = os.Stat(filepath.Join(out, "Contents", "MacOS", HelperName))
	require.Error(t, err)
	plist, err := os.ReadFile(filepath.Join(out, "Contents", "Info.plist"))
	require.NoError(t, err)
	require.Contains(t, string(plist), "<key>CFBundleExecutable</key>\n\t<string>Demo</string>")
	_, err = os.Stat(filepath.Join(out, "Contents", "Resources", "AppIcon.icns"))
	require.NoError(t, err)
	pkg, err := os.ReadFile(filepath.Join(out, "Contents", "PkgInfo"))
	require.NoError(t, err)
	require.Equal(t, "APPL????", string(pkg))
}
