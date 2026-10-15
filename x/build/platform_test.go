package build

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGOOSEnum(t *testing.T) {
	require.Equal(t, GOOS(runtime.GOOS), HostGOOS())
	require.True(t, HostGOOS().IsValid())

	for _, name := range []string{"linux", "darwin", "windows", "android", "ios", "freebsd", "wasip1", "js", "linux_iso", "linux_uki", "mbr", "uefi"} {
		os, err := ParseGOOS(name)
		require.NoError(t, err)
		require.Equal(t, name, os.String())
		require.True(t, os.IsValid())
	}

	_, err := ParseGOOS("bogus")
	require.ErrorIs(t, err, ErrGOOS)

	require.True(t, GOOSLinuxISO.Guest())
	require.True(t, GOOSLinuxUKI.Guest())
	require.False(t, GOOSLinux.Guest())

	require.True(t, GOOSUEFI.Firmware())
	require.True(t, GOOSMBR.Firmware())
	require.False(t, GOOSLinux.Firmware())

	require.True(t, GOOSAndroid.HasAppPackage())
	require.True(t, GOOSDarwin.HasAppPackage())
	require.True(t, GOOSIOS.HasAppPackage())
	require.True(t, GOOSLinux.HasAppPackage())
	require.True(t, GOOSWindows.HasAppPackage())
	require.False(t, GOOSPlan9.HasAppPackage())

	require.True(t, GOOSAndroid.Scaffolded())
	require.True(t, GOOSDarwin.Scaffolded())
	require.True(t, GOOSIOS.Scaffolded())
	require.False(t, GOOSLinux.Scaffolded())
}

func TestGOARCHEnum(t *testing.T) {
	require.Equal(t, GOARCH(runtime.GOARCH), HostGOARCH())
	require.True(t, HostGOARCH().IsValid())

	cases := map[string]GOARCH{
		"amd64":   GOARCHAMD64,
		"x86_64":  GOARCHAMD64,
		"x64":     GOARCHAMD64,
		"arm64":   GOARCHARM64,
		"aarch64": GOARCHARM64,
		"386":     GOARCH386,
		"x86":     GOARCH386,
		"i386":    GOARCH386,
		"arm":     GOARCHARM,
		"armv7l":  GOARCHARM,
		"wasm":    GOARCHWasm,
		"wasm32":  GOARCHWasm,
	}
	for input, want := range cases {
		got, err := ParseGOARCH(input)
		require.NoError(t, err, input)
		require.Equal(t, want, got, input)
	}

	_, err := ParseGOARCH("bogus")
	require.ErrorIs(t, err, ErrGOARCH)

	abi, err := GOARCHARM64.AndroidABI()
	require.NoError(t, err)
	require.Equal(t, ABIARM64V8A, abi)
	require.Equal(t, GOARCHARM64, abi.GOARCH())

	_, err = GOARCHPPC64.AndroidABI()
	require.ErrorIs(t, err, ErrABI)
}

func TestPlatformUberEnum(t *testing.T) {
	host := HostPlatform()
	require.True(t, host.IsValid())
	require.Equal(t, HostGOOS(), host.GOOS())
	require.Equal(t, HostGOARCH(), host.GOARCH())

	for _, p := range Platform("").Values() {
		require.True(t, p.IsValid())
		goos, goarch := p.Split()
		require.True(t, goos.IsValid(), p.String())
		require.True(t, goarch.IsValid(), p.String())

		roundTrip, err := PlatformFor(goos, goarch)
		require.NoError(t, err, p.String())
		require.Equal(t, p, roundTrip)
		require.Equal(t, Target{GOOS: goos.String(), GOARCH: goarch.String()}, p.Target())
	}

	aliases := map[string]Platform{
		"linux/amd64":         PlatformLinuxAMD64,
		"linux_amd64":         PlatformLinuxAMD64,
		"linux-64":            PlatformLinuxAMD64,
		"linux/arm64":         PlatformLinuxARM64,
		"linux-aarch64":       PlatformLinuxARM64,
		"darwin/arm64":        PlatformDarwinARM64,
		"osx-arm64":           PlatformDarwinARM64,
		"windows/amd64":       PlatformWindowsAMD64,
		"win-64":              PlatformWindowsAMD64,
		"android/arm64":       PlatformAndroidARM64,
		"android/arm64-v8a":   PlatformAndroidARM64,
		"android_arm64-v8a":   PlatformAndroidARM64,
		"android/armeabi-v7a": PlatformAndroidARM,
		"android/x86_64":      PlatformAndroidAMD64,
		"android/x86":         PlatformAndroid386,
		"linux_iso/amd64":     PlatformLinuxISOAMD64,
		"linux_uki/arm64":     PlatformLinuxUKIARM64,
		"uefi/amd64":          PlatformUEFIAMD64,
		"mbr/amd64":           PlatformMBRAMD64,
	}
	for input, want := range aliases {
		got, err := ParsePlatform(input)
		require.NoError(t, err, input)
		require.Equal(t, want, got, input)
	}

	_, err := ParsePlatform("plan9/wasm")
	require.ErrorIs(t, err, ErrPlatform)

	_, err = PlatformFor(GOOSPlan9, GOARCHWasm)
	require.ErrorIs(t, err, ErrPlatform)

	abi, err := PlatformAndroidARM64.AndroidABI()
	require.NoError(t, err)
	require.Equal(t, ABIARM64V8A, abi)

	_, err = PlatformLinuxAMD64.AndroidABI()
	require.ErrorIs(t, err, ErrABI)

	require.Len(t, DesktopPlatforms(), 6)
	platforms, err := HostPlatforms(t.Context())
	require.NoError(t, err)
	require.Contains(t, platforms, PlatformLinuxAMD64)
}
