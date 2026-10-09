package main

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathUsesFileInsideDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath("android", "arm64", dir, build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "contapila_android_arm64.apk"), got)
}

func TestArtifactPathMissingDirectory(t *testing.T) {
	got, err := artifactPath("android", "arm64", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "contapila_android_arm64.apk"), got)
}

func TestArtifactPathWindowsExe(t *testing.T) {
	got, err := artifactPath("windows", "amd64", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_windows_amd64.exe"), got)
}

func TestArtifactPathDarwinApp(t *testing.T) {
	got, err := artifactPath("darwin", "arm64", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_darwin_arm64.app"), got)
}

func TestArtifactPathLinuxAppImage(t *testing.T) {
	got, err := artifactPath("linux", "amd64", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.drivers", Name: "Drivers"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Drivers_linux_amd64.AppImage"), got)
}

func TestArtifactPathKeepsExplicitFile(t *testing.T) {
	got, err := artifactPath("android", "arm64", "out/app.apk", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, "out/app.apk", got)
}
