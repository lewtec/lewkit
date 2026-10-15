package build

import (
	"fmt"
	"runtime"
	"strings"
)

// GOOS is a Go or Lewkit target operating system.
//
//lewkit:enum
type GOOS string

const (
	GOOSAix       GOOS = "aix"
	GOOSAndroid   GOOS = "android"
	GOOSDarwin    GOOS = "darwin"
	GOOSDragonfly GOOS = "dragonfly"
	GOOSFreebsd   GOOS = "freebsd"
	GOOSIllumos   GOOS = "illumos"
	GOOSIOS       GOOS = "ios"
	GOOSJS        GOOS = "js"
	GOOSLinux     GOOS = "linux"
	GOOSLinuxISO  GOOS = "linux_iso"
	GOOSLinuxUKI  GOOS = "linux_uki"
	GOOSMBR       GOOS = "mbr"
	GOOSNetbsd    GOOS = "netbsd"
	GOOSOpenbsd   GOOS = "openbsd"
	GOOSPlan9     GOOS = "plan9"
	GOOSSolaris   GOOS = "solaris"
	GOOSTamago    GOOS = "tamago"
	GOOSUEFI      GOOS = "uefi"
	GOOSWasip1    GOOS = "wasip1"
	GOOSWindows   GOOS = "windows"
)

// HostGOOS returns the process GOOS.
func HostGOOS() GOOS {
	return GOOS(runtime.GOOS)
}

// Guest reports whether os is a Linux guest image (linux_uki or linux_iso).
func (os GOOS) Guest() bool {
	return os == GOOSLinuxUKI || os == GOOSLinuxISO
}

// Firmware reports whether os is a unikernel firmware image (uefi or mbr).
func (os GOOS) Firmware() bool {
	return os == GOOSUEFI || os == GOOSMBR
}

// HasAppPackage reports whether os has a packaged host app format.
func (os GOOS) HasAppPackage() bool {
	switch os {
	case GOOSDarwin, GOOSAndroid, GOOSIOS, GOOSWindows, GOOSLinux:
		return true
	default:
		return false
	}
}

// Scaffolded reports whether packaging a host app for os scaffolds a host project tree.
func (os GOOS) Scaffolded() bool {
	switch os {
	case GOOSDarwin, GOOSAndroid, GOOSIOS:
		return true
	default:
		return false
	}
}

// GOARCH is a Go target architecture.
//
//lewkit:enum
type GOARCH string

const (
	GOARCH386      GOARCH = "386"     // x86, i386, i686
	GOARCHAMD64    GOARCH = "amd64"   // x86_64, x64
	GOARCHARM      GOARCH = "arm"     // armv7l, armeabi-v7a
	GOARCHARM64    GOARCH = "arm64"   // aarch64, arm64-v8a
	GOARCHLoong64  GOARCH = "loong64" // loongarch64
	GOARCHMips     GOARCH = "mips"
	GOARCHMips64   GOARCH = "mips64"
	GOARCHMips64le GOARCH = "mips64le"
	GOARCHMipsle   GOARCH = "mipsle"
	GOARCHPPC64    GOARCH = "ppc64"
	GOARCHPPC64le  GOARCH = "ppc64le"
	GOARCHRiscv64  GOARCH = "riscv64"
	GOARCHS390x    GOARCH = "s390x"
	GOARCHSparc64  GOARCH = "sparc64"
	GOARCHWasm     GOARCH = "wasm" // wasm32
)

// HostGOARCH returns the process GOARCH.
func HostGOARCH() GOARCH {
	return GOARCH(runtime.GOARCH)
}

// AndroidABI returns the Android ABI for arch.
func (arch GOARCH) AndroidABI() (ABI, error) {
	return ABIFor(arch)
}

// Platform is the uber enum for valid combinations of GOOS, GOARCH, and target aliases.
//
//lewkit:enum
type Platform string

const (
	PlatformAixPPC64        Platform = "aix/ppc64"        // aix_ppc64
	PlatformAndroid386      Platform = "android/386"      // android_386, android/x86, android_x86
	PlatformAndroidAMD64    Platform = "android/amd64"    // android_amd64, android/x86_64, android_x86_64
	PlatformAndroidARM      Platform = "android/arm"      // android_arm, android/armeabi-v7a, android_armeabi-v7a
	PlatformAndroidARM64    Platform = "android/arm64"    // android_arm64, android/arm64-v8a, android_arm64-v8a
	PlatformDarwinAMD64     Platform = "darwin/amd64"     // darwin_amd64, osx-64
	PlatformDarwinARM64     Platform = "darwin/arm64"     // darwin_arm64, osx-arm64
	PlatformDragonflyAMD64  Platform = "dragonfly/amd64"  // dragonfly_amd64
	PlatformFreebsd386      Platform = "freebsd/386"      // freebsd_386
	PlatformFreebsdAMD64    Platform = "freebsd/amd64"    // freebsd_amd64
	PlatformFreebsdARM      Platform = "freebsd/arm"      // freebsd_arm
	PlatformFreebsdARM64    Platform = "freebsd/arm64"    // freebsd_arm64
	PlatformFreebsdRiscv64  Platform = "freebsd/riscv64"  // freebsd_riscv64
	PlatformIllumosAMD64    Platform = "illumos/amd64"    // illumos_amd64
	PlatformIOSAMD64        Platform = "ios/amd64"        // ios_amd64
	PlatformIOSARM64        Platform = "ios/arm64"        // ios_arm64
	PlatformJSWasm          Platform = "js/wasm"          // js_wasm
	PlatformLinux386        Platform = "linux/386"        // linux_386, linux-32
	PlatformLinuxAMD64      Platform = "linux/amd64"      // linux_amd64, linux-64
	PlatformLinuxARM        Platform = "linux/arm"        // linux_arm
	PlatformLinuxARM64      Platform = "linux/arm64"      // linux_arm64, linux-aarch64
	PlatformLinuxLoong64    Platform = "linux/loong64"    // linux_loong64
	PlatformLinuxMips       Platform = "linux/mips"       // linux_mips
	PlatformLinuxMips64     Platform = "linux/mips64"     // linux_mips64
	PlatformLinuxMips64le   Platform = "linux/mips64le"   // linux_mips64le
	PlatformLinuxMipsle     Platform = "linux/mipsle"     // linux_mipsle
	PlatformLinuxPPC64      Platform = "linux/ppc64"      // linux_ppc64
	PlatformLinuxPPC64le    Platform = "linux/ppc64le"    // linux_ppc64le, linux-ppc64le
	PlatformLinuxRiscv64    Platform = "linux/riscv64"    // linux_riscv64, linux-riscv64
	PlatformLinuxS390x      Platform = "linux/s390x"      // linux_s390x, linux-s390x
	PlatformLinuxSparc64    Platform = "linux/sparc64"    // linux_sparc64
	PlatformLinuxISOAMD64   Platform = "linux_iso/amd64"  // linux_iso_amd64
	PlatformLinuxISOARM64   Platform = "linux_iso/arm64"  // linux_iso_arm64
	PlatformLinuxUKIAMD64   Platform = "linux_uki/amd64"  // linux_uki_amd64
	PlatformLinuxUKIARM64   Platform = "linux_uki/arm64"  // linux_uki_arm64
	PlatformMBRAMD64        Platform = "mbr/amd64"        // mbr_amd64
	PlatformNetbsd386       Platform = "netbsd/386"       // netbsd_386
	PlatformNetbsdAMD64     Platform = "netbsd/amd64"     // netbsd_amd64
	PlatformNetbsdARM       Platform = "netbsd/arm"       // netbsd_arm
	PlatformNetbsdARM64     Platform = "netbsd/arm64"     // netbsd_arm64
	PlatformOpenbsd386      Platform = "openbsd/386"      // openbsd_386
	PlatformOpenbsdAMD64    Platform = "openbsd/amd64"    // openbsd_amd64
	PlatformOpenbsdARM      Platform = "openbsd/arm"      // openbsd_arm
	PlatformOpenbsdARM64    Platform = "openbsd/arm64"    // openbsd_arm64
	PlatformOpenbsdPPC64    Platform = "openbsd/ppc64"    // openbsd_ppc64
	PlatformOpenbsdRiscv64  Platform = "openbsd/riscv64"  // openbsd_riscv64
	PlatformPlan9386        Platform = "plan9/386"        // plan9_386
	PlatformPlan9AMD64      Platform = "plan9/amd64"      // plan9_amd64
	PlatformPlan9ARM        Platform = "plan9/arm"        // plan9_arm
	PlatformSolarisAMD64    Platform = "solaris/amd64"    // solaris_amd64
	PlatformTamagoAMD64     Platform = "tamago/amd64"     // tamago_amd64
	PlatformTamagoARM       Platform = "tamago/arm"       // tamago_arm
	PlatformTamagoARM64     Platform = "tamago/arm64"     // tamago_arm64
	PlatformTamagoRiscv64   Platform = "tamago/riscv64"   // tamago_riscv64
	PlatformUEFIAMD64       Platform = "uefi/amd64"       // uefi_amd64
	PlatformWasip1Wasm      Platform = "wasip1/wasm"      // wasip1_wasm
	PlatformWindows386      Platform = "windows/386"      // windows_386, win-32
	PlatformWindowsAMD64    Platform = "windows/amd64"    // windows_amd64, win-64
	PlatformWindowsARM64    Platform = "windows/arm64"    // windows_arm64, win-arm64
)

// HostPlatform returns the process Platform.
func HostPlatform() Platform {
	return Platform(runtime.GOOS + "/" + runtime.GOARCH)
}

// PlatformFor resolves the Platform for goos and goarch.
func PlatformFor(goos GOOS, goarch GOARCH) (Platform, error) {
	return ParsePlatform(string(goos) + "/" + string(goarch))
}

// Split returns the GOOS and GOARCH components of p.
func (p Platform) Split() (GOOS, GOARCH) {
	os, arch, ok := strings.Cut(string(p), "/")
	if !ok {
		return "", ""
	}
	return GOOS(os), GOARCH(arch)
}

// GOOS returns the operating system component of p.
func (p Platform) GOOS() GOOS {
	os, _ := p.Split()
	return os
}

// GOARCH returns the architecture component of p.
func (p Platform) GOARCH() GOARCH {
	_, arch := p.Split()
	return arch
}

// Target converts p to a Target.
func (p Platform) Target() Target {
	os, arch := p.Split()
	return Target{GOOS: string(os), GOARCH: string(arch)}
}

// AndroidABI returns the Android ABI for p.
func (p Platform) AndroidABI() (ABI, error) {
	if p.GOOS() != GOOSAndroid {
		return "", fmt.Errorf("%w: %s", ErrABI, p)
	}
	return ABIFor(p.GOARCH())
}

// DesktopPlatforms is the release matrix: Linux, macOS, and Windows on amd64 and arm64.
func DesktopPlatforms() []Platform {
	return []Platform{
		PlatformLinuxAMD64,
		PlatformLinuxARM64,
		PlatformDarwinAMD64,
		PlatformDarwinARM64,
		PlatformWindowsAMD64,
		PlatformWindowsARM64,
	}
}
