package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostTargetsSkipUnavailableHosts(t *testing.T) {
	targets, err := HostTargets(t.Context())
	require.NoError(t, err)
	require.Contains(t, targets, Target{GOOS: "linux", GOARCH: "amd64"})
	require.Contains(t, targets, Target{GOOS: "linux", GOARCH: "arm64"})
	require.Contains(t, targets, Target{GOOS: "windows", GOARCH: "amd64"})
	require.Contains(t, targets, Target{GOOS: "windows", GOARCH: "arm64"})
	if runtime.GOOS != "darwin" {
		for _, target := range targets {
			require.NotEqual(t, "darwin", target.GOOS)
			require.NotEqual(t, "ios", target.GOOS)
		}
	}
}

func TestUberGraphSharesIcons(t *testing.T) {
	dir := t.TempDir()
	out := t.TempDir()
	u := Uber{
		Spec: Spec{Dir: dir, ID: "br.tec.lew.demo", Name: "Demo"},
		Out:  out,
		Name: "demo",
	}
	apps := []Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "android", GOARCH: "arm64"},
	}
	graph, paths, _, err := u.assemble(DesktopTargets(), apps)
	require.NoError(t, err)
	steps := map[string][]string{}
	for _, step := range graph.Steps {
		steps[step.Name] = step.Deps
	}
	deps, ok := steps["linux/amd64"]
	require.True(t, ok)
	require.Empty(t, deps)
	deps, ok = steps["icons"]
	require.True(t, ok)
	require.Empty(t, deps)
	require.Equal(t, []string{"icons"}, steps["app/linux/amd64"])
	require.Equal(t, []string{"icons"}, steps["app/windows/arm64"])
	require.NotContains(t, steps, "scaffold/linux/amd64")
	require.Equal(t, []string{"icons"}, steps["scaffold/darwin/arm64"])
	require.Equal(t, []string{"scaffold/darwin/arm64"}, steps["app/darwin/arm64"])
	require.Equal(t, []string{"icons"}, steps["scaffold/android/arm64"])
	require.Equal(t, []string{"scaffold/android/arm64"}, steps["app/android/arm64"])
	require.Contains(t, graph.Defaults, "linux/amd64")
	require.Contains(t, graph.Defaults, "app/linux/amd64")
	require.NotContains(t, graph.Defaults, "icons")
	require.NotContains(t, graph.Defaults, "scaffold/darwin/arm64")
	require.Contains(t, paths, filepath.Join(out, "demo_darwin_arm64.tar.gz"))
	require.Contains(t, paths, filepath.Join(out, "Demo_linux_amd64.AppImage"))
	require.Contains(t, paths, filepath.Join(out, "Demo_windows_arm64.exe"))
	require.Contains(t, paths, filepath.Join(out, "demo_android_arm64.apk"))
}

func TestUberRejectsFileOut(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dist")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	u := Uber{Spec: Spec{Dir: dir, ID: "br.tec.lew.demo", Name: "Demo"}, Out: file}
	_, _, _, err := u.assemble(DesktopTargets(), nil)
	require.Error(t, err)
}

func TestUberWritesDist(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := t.TempDir()
	paths, err := Uber{
		Spec: Spec{Dir: mainDir, ID: "br.tec.lew.demo", Name: "Demo", Version: "1.2.3"},
		Out:  out,
		Name: "demo",
	}.Run(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0))
	}
	raw, err := os.ReadFile(filepath.Join(out, "Demo_linux_amd64.AppImage"))
	require.NoError(t, err)
	require.Equal(t, "\x7fELF", string(raw[:4]))
	zipRaw, err := os.ReadFile(filepath.Join(out, "demo_windows_amd64.zip"))
	require.NoError(t, err)
	require.Equal(t, "PK", string(zipRaw[:2]))
}
