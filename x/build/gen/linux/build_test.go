package linux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
	"github.com/stretchr/testify/require"
)

func TestBuildNilContextPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			require.FailNow(t, fmt.Sprint("nil context returned"))
		}
	}()
	_, _ = Build(nil, BuildOptions{})
}

func TestBuildAppImage(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := filepath.Join(t.TempDir(), "Demo.AppImage")
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
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	if result.AppPath != out {
		require.FailNowf(t, "", "app %s", result.AppPath)
	}
	info, err := os.Stat(out)
	require.NoError(t, err)
	if info.IsDir() {
		require.FailNow(t, fmt.Sprint("app image is a directory"))
	}
	if info.Mode()&0o111 == 0 {
		require.FailNow(t, fmt.Sprint("app image is not executable"))
	}
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	if string(raw[:4]) != "\x7fELF" {
		require.FailNowf(t, "", "prefix %q", raw[:4])
	}
	tr, err := release.OpenTrailer(out)
	require.NoError(t, err)
	defer tr.Close()
	marker, err := tr.Bytes(release.MarkerFile)
	require.NoError(t, err)
	if strings.TrimSpace(string(marker)) != "br.tec.lew.demo" {
		require.FailNowf(t, "", "marker %q", marker)
	}
	desktop, err := tr.Bytes("Demo.desktop")
	require.NoError(t, err)
	text := string(desktop)
	if !strings.Contains(text, "StartupWMClass=br.tec.lew.demo") || !strings.Contains(text, "Exec=Demo") || !strings.Contains(text, "Icon=icon") {
		require.FailNowf(t, "", "desktop %s", text)
	}
	icon, err := tr.Bytes("icon.png")
	if err != nil || len(icon) == 0 {
		require.FailNowf(t, "", "icon %v", err)
	}
}
