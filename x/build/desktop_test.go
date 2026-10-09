package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/stretchr/testify/require"
)

func TestArchiveName(t *testing.T) {
	require.Equal(t, "demo_linux_amd64.tar.gz", ArchiveName("demo", Target{GOOS: "linux", GOARCH: "amd64"}))
	require.Equal(t, "demo_windows_arm64.zip", ArchiveName("demo", Target{GOOS: "windows", GOARCH: "arm64"}))
	require.Len(t, DesktopTargets(), 6)
}

func TestAppFile(t *testing.T) {
	require.Equal(t, "Demo_windows_amd64.exe", AppFile("Demo", "br.tec.lew.demo", "windows", "amd64"))
	require.Equal(t, "Demo_darwin_arm64.app", AppFile("Demo", "br.tec.lew.demo", "darwin", "arm64"))
	require.Equal(t, "Demo_ios_arm64.app", AppFile("Demo", "br.tec.lew.demo", "ios", "arm64"))
	require.Equal(t, "Drivers_linux_amd64.AppImage", AppFile("Drivers", "br.tec.lew.drivers", "linux", "amd64"))
	require.Equal(t, "contapila_android_arm64.apk", AppFile("Ignored", "br.tec.lew.contapila", "android", "arm64"))
	require.Equal(t, "app_android_amd64.apk", AppFile("Demo", "", "android", "amd64"))
}

func TestJobPaths(t *testing.T) {
	job := Job{
		Out:     "dist",
		Name:    "demo",
		Targets: DesktopTargets(),
		Sign:    &sign.Identity{},
	}
	require.Equal(t, []string{
		filepath.Join("dist", "demo_linux_amd64.tar.gz"),
		filepath.Join("dist", "demo_linux_amd64.tar.gz.cms"),
		filepath.Join("dist", "demo_linux_arm64.tar.gz"),
		filepath.Join("dist", "demo_linux_arm64.tar.gz.cms"),
		filepath.Join("dist", "demo_darwin_amd64.tar.gz"),
		filepath.Join("dist", "demo_darwin_amd64.tar.gz.cms"),
		filepath.Join("dist", "demo_darwin_arm64.tar.gz"),
		filepath.Join("dist", "demo_darwin_arm64.tar.gz.cms"),
		filepath.Join("dist", "demo_windows_amd64.zip"),
		filepath.Join("dist", "demo_windows_amd64.zip.cms"),
		filepath.Join("dist", "demo_windows_arm64.zip"),
		filepath.Join("dist", "demo_windows_arm64.zip.cms"),
	}, job.Paths())
}

func TestJobWritesArchive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := t.TempDir()
	target := Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	paths, err := Job{
		Dir:     dir,
		Out:     out,
		Name:    "demo",
		Targets: []Target{target},
	}.Run(t.Context())
	require.NoError(t, err)
	name := ArchiveName("demo", target)
	require.Equal(t, []string{filepath.Join(out, name)}, paths)
	member := archiveMember(t, paths[0])
	want := "demo"
	if runtime.GOOS == "windows" {
		want = "demo.exe"
	}
	require.Equal(t, want, member)
}

func archiveMember(t *testing.T, path string) string {
	t.Helper()
	if strings.HasSuffix(path, ".zip") {
		reader, err := zip.OpenReader(path)
		require.NoError(t, err)
		defer reader.Close()
		require.NotEmpty(t, reader.File)
		return reader.File[0].Name
	}
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	gz, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gz.Close()
	header, err := tar.NewReader(gz).Next()
	require.NoError(t, err)
	return header.Name
}
