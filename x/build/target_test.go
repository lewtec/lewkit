package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveName(t *testing.T) {
	require.Equal(t, "demo_linux_amd64.tar.gz", ArchiveName("demo", Target{GOOS: "linux", GOARCH: "amd64"}))
	require.Equal(t, "demo_windows_arm64.zip", ArchiveName("demo", Target{GOOS: "windows", GOARCH: "arm64"}))
	require.Equal(t, "demo_uefi_amd64.efi", ArchiveName("demo", Target{GOOS: "uefi", GOARCH: "amd64"}))
	require.Equal(t, "demo_mbr_amd64.img", ArchiveName("demo", Target{GOOS: "mbr", GOARCH: "amd64"}))
	require.Equal(t, "demo_linux_uki_amd64.efi", ArchiveName("demo", Target{GOOS: "linux_uki", GOARCH: "amd64"}))
	require.Equal(t, "demo_linux_iso_amd64.iso", ArchiveName("demo", Target{GOOS: "linux_iso", GOARCH: "amd64"}))
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
