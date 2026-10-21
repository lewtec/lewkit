package main

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/host"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathUsesFileInsideDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath(t.Context(), "android", "arm64", dir, host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "contapila_android_arm64.apk"), got)
}

func TestArtifactPathMissingDirectory(t *testing.T) {
	got, err := artifactPath(t.Context(), "android", "arm64", "dist", host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "contapila_android_arm64.apk"), got)
}

func TestArtifactPathWindowsExe(t *testing.T) {
	got, err := artifactPath(t.Context(), "windows", "amd64", "dist", host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_windows_amd64.exe"), got)
}

func TestArtifactPathDarwinApp(t *testing.T) {
	got, err := artifactPath(t.Context(), "darwin", "arm64", "dist", host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_darwin_arm64.app"), got)
}

func TestArtifactPathLinuxAppImage(t *testing.T) {
	got, err := artifactPath(t.Context(), "linux", "amd64", "dist", host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.drivers", Name: "Drivers"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Drivers_linux_amd64.AppImage"), got)
}

func TestArtifactPathKeepsExplicitFile(t *testing.T) {
	got, err := artifactPath(t.Context(), "android", "arm64", "out/app.apk", host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, "out/app.apk", got)
}
