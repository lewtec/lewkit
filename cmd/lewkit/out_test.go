package main

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/host"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathUsesFileInsideDirectory(t *testing.T) {
	dir := t.TempDir()
	flags := (artifactInput{GOOS: "android", GOARCH: "arm64", Out: dir, Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "contapila_android_arm64.apk"), got)
}

func TestArtifactPathMissingDirectory(t *testing.T) {
	flags := (artifactInput{GOOS: "android", GOARCH: "arm64", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "contapila_android_arm64.apk"), got)
}

func TestArtifactPathWindowsExe(t *testing.T) {
	flags := (artifactInput{GOOS: "windows", GOARCH: "amd64", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_windows_amd64.exe"), got)
}

func TestArtifactPathDarwinApp(t *testing.T) {
	flags := (artifactInput{GOOS: "darwin", GOARCH: "arm64", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_darwin_arm64.app"), got)
}

func TestArtifactPathLinuxAppImage(t *testing.T) {
	flags := (artifactInput{GOOS: "linux", GOARCH: "amd64", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.drivers", Name: "Drivers"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Drivers_linux_amd64.AppImage"), got)
}

func TestArtifactPathKeepsExplicitFile(t *testing.T) {
	flags := (artifactInput{GOOS: "android", GOARCH: "arm64", Out: "out/app.apk", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, "out/app.apk", got)
}

func TestArtifactPathIOSSimulatorApp(t *testing.T) {
	flags := (artifactInput{GOOS: "ios", GOARCH: "arm64", SDK: "iphonesimulator", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_ios_arm64.app"), got)
}

func TestArtifactPathIOSDeviceIPA(t *testing.T) {
	flags := (artifactInput{GOOS: "ios", GOARCH: "arm64", SDK: "iphoneos", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"}}).flags(t)
	got, err := flags.artifactPath(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "Counter_ios_arm64.ipa"), got)
}

func TestArtifactPathIOSRejectsSDK(t *testing.T) {
	flags := (artifactInput{GOOS: "ios", GOARCH: "arm64", SDK: "watchos", Out: "dist", Spec: host.Spec{Dir: t.TempDir(), ID: "br.tec.lew.counter", Name: "Counter"}}).flags(t)
	_, err := flags.artifactPath(t.Context())
	require.ErrorIs(t, err, ios.ErrSDK)
}

type artifactInput struct {
	GOOS   string
	GOARCH string
	SDK    string
	Out    string
	Spec   host.Spec
}

func (in artifactInput) flags(t *testing.T) buildFlags {
	t.Helper()
	var flags buildFlags
	require.NoError(t, flags.goos.Parse(in.GOOS))
	require.NoError(t, flags.goarch.Parse(in.GOARCH))
	if in.SDK != "" {
		require.NoError(t, flags.sdk.Parse(in.SDK))
	}
	require.NoError(t, flags.out.Parse(in.Out))
	require.NoError(t, flags.dir.Parse(in.Spec.Dir))
	if in.Spec.ID != "" {
		require.NoError(t, flags.id.Parse(in.Spec.ID))
	}
	if in.Spec.Name != "" {
		require.NoError(t, flags.name.Parse(in.Spec.Name))
	}
	return flags
}
