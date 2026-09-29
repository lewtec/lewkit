package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathUsesFileInsideDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath("android", dir, build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "contapila-debug.apk"), got)
}

func TestArtifactPathMissingDirectory(t *testing.T) {
	got, err := artifactPath("android", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "contapila-debug.apk"), got)
}

func TestArtifactPathKeepsExplicitFile(t *testing.T) {
	got, err := artifactPath("android", "out/app.apk", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, "out/app.apk", got)
}

func TestArtifactPathWindowsExe(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath("windows", dir, build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila", Name: "Conta Pila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "Conta-Pila.exe"), got)
}

func TestArtifactPathLinuxBinary(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath("linux", dir, build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila", Name: "Conta Pila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "Conta-Pila"), got)
}

func TestArtifactPathKeepsExplicitWindowsFile(t *testing.T) {
	got, err := artifactPath("windows", "out/Basic.exe", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila", Name: "Basic"})
	require.NoError(t, err)
	require.Equal(t, "out/Basic.exe", got)
}

func TestLaunchWindowsReportsThisMachine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	err := launchApp(t.Context(), "windows", "dist/Basic.exe", "br.tec.lew.basic")
	require.EqualError(t, err, "built windows (dist/Basic.exe); this machine is "+runtime.GOOS)
}
