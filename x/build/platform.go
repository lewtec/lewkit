package build

import (
	"fmt"
	"runtime"
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

// Platform is one GOOS and one GOARCH.
//
//lewkit:enum compose=GOOS/GOARCH
type Platform string

const (
	PlatformAixPPC64       = Platform(string(GOOSAix) + "/" + string(GOARCHPPC64))
	PlatformAndroid386     = Platform(string(GOOSAndroid) + "/" + string(GOARCH386))
	PlatformAndroidAMD64   = Platform(string(GOOSAndroid) + "/" + string(GOARCHAMD64))
	PlatformAndroidARM     = Platform(string(GOOSAndroid) + "/" + string(GOARCHARM))
	PlatformAndroidARM64   = Platform(string(GOOSAndroid) + "/" + string(GOARCHARM64))
	PlatformDarwinAMD64    = Platform(string(GOOSDarwin) + "/" + string(GOARCHAMD64)) // osx-64
	PlatformDarwinARM64    = Platform(string(GOOSDarwin) + "/" + string(GOARCHARM64)) // osx-arm64
	PlatformDragonflyAMD64 = Platform(string(GOOSDragonfly) + "/" + string(GOARCHAMD64))
	PlatformFreebsd386     = Platform(string(GOOSFreebsd) + "/" + string(GOARCH386))
	PlatformFreebsdAMD64   = Platform(string(GOOSFreebsd) + "/" + string(GOARCHAMD64))
	PlatformFreebsdARM     = Platform(string(GOOSFreebsd) + "/" + string(GOARCHARM))
	PlatformFreebsdARM64   = Platform(string(GOOSFreebsd) + "/" + string(GOARCHARM64))
	PlatformFreebsdRiscv64 = Platform(string(GOOSFreebsd) + "/" + string(GOARCHRiscv64))
	PlatformIllumosAMD64   = Platform(string(GOOSIllumos) + "/" + string(GOARCHAMD64))
	PlatformIOSAMD64       = Platform(string(GOOSIOS) + "/" + string(GOARCHAMD64))
	PlatformIOSARM64       = Platform(string(GOOSIOS) + "/" + string(GOARCHARM64))
	PlatformJSWasm         = Platform(string(GOOSJS) + "/" + string(GOARCHWasm))
	PlatformLinux386       = Platform(string(GOOSLinux) + "/" + string(GOARCH386))   // linux-32
	PlatformLinuxAMD64     = Platform(string(GOOSLinux) + "/" + string(GOARCHAMD64)) // linux-64
	PlatformLinuxARM       = Platform(string(GOOSLinux) + "/" + string(GOARCHARM))
	PlatformLinuxARM64     = Platform(string(GOOSLinux) + "/" + string(GOARCHARM64)) // linux-aarch64
	PlatformLinuxLoong64   = Platform(string(GOOSLinux) + "/" + string(GOARCHLoong64))
	PlatformLinuxMips      = Platform(string(GOOSLinux) + "/" + string(GOARCHMips))
	PlatformLinuxMips64    = Platform(string(GOOSLinux) + "/" + string(GOARCHMips64))
	PlatformLinuxMips64le  = Platform(string(GOOSLinux) + "/" + string(GOARCHMips64le))
	PlatformLinuxMipsle    = Platform(string(GOOSLinux) + "/" + string(GOARCHMipsle))
	PlatformLinuxPPC64     = Platform(string(GOOSLinux) + "/" + string(GOARCHPPC64))
	PlatformLinuxPPC64le   = Platform(string(GOOSLinux) + "/" + string(GOARCHPPC64le)) // linux-ppc64le
	PlatformLinuxRiscv64   = Platform(string(GOOSLinux) + "/" + string(GOARCHRiscv64)) // linux-riscv64
	PlatformLinuxS390x     = Platform(string(GOOSLinux) + "/" + string(GOARCHS390x))   // linux-s390x
	PlatformLinuxSparc64   = Platform(string(GOOSLinux) + "/" + string(GOARCHSparc64))
	PlatformLinuxISOAMD64  = Platform(string(GOOSLinuxISO) + "/" + string(GOARCHAMD64))
	PlatformLinuxISOARM64  = Platform(string(GOOSLinuxISO) + "/" + string(GOARCHARM64))
	PlatformLinuxUKIAMD64  = Platform(string(GOOSLinuxUKI) + "/" + string(GOARCHAMD64))
	PlatformLinuxUKIARM64  = Platform(string(GOOSLinuxUKI) + "/" + string(GOARCHARM64))
	PlatformMBRAMD64       = Platform(string(GOOSMBR) + "/" + string(GOARCHAMD64))
	PlatformNetbsd386      = Platform(string(GOOSNetbsd) + "/" + string(GOARCH386))
	PlatformNetbsdAMD64    = Platform(string(GOOSNetbsd) + "/" + string(GOARCHAMD64))
	PlatformNetbsdARM      = Platform(string(GOOSNetbsd) + "/" + string(GOARCHARM))
	PlatformNetbsdARM64    = Platform(string(GOOSNetbsd) + "/" + string(GOARCHARM64))
	PlatformOpenbsd386     = Platform(string(GOOSOpenbsd) + "/" + string(GOARCH386))
	PlatformOpenbsdAMD64   = Platform(string(GOOSOpenbsd) + "/" + string(GOARCHAMD64))
	PlatformOpenbsdARM     = Platform(string(GOOSOpenbsd) + "/" + string(GOARCHARM))
	PlatformOpenbsdARM64   = Platform(string(GOOSOpenbsd) + "/" + string(GOARCHARM64))
	PlatformOpenbsdPPC64   = Platform(string(GOOSOpenbsd) + "/" + string(GOARCHPPC64))
	PlatformOpenbsdRiscv64 = Platform(string(GOOSOpenbsd) + "/" + string(GOARCHRiscv64))
	PlatformPlan9386       = Platform(string(GOOSPlan9) + "/" + string(GOARCH386))
	PlatformPlan9AMD64     = Platform(string(GOOSPlan9) + "/" + string(GOARCHAMD64))
	PlatformPlan9ARM       = Platform(string(GOOSPlan9) + "/" + string(GOARCHARM))
	PlatformSolarisAMD64   = Platform(string(GOOSSolaris) + "/" + string(GOARCHAMD64))
	PlatformTamagoAMD64    = Platform(string(GOOSTamago) + "/" + string(GOARCHAMD64))
	PlatformTamagoARM      = Platform(string(GOOSTamago) + "/" + string(GOARCHARM))
	PlatformTamagoARM64    = Platform(string(GOOSTamago) + "/" + string(GOARCHARM64))
	PlatformTamagoRiscv64  = Platform(string(GOOSTamago) + "/" + string(GOARCHRiscv64))
	PlatformUEFIAMD64      = Platform(string(GOOSUEFI) + "/" + string(GOARCHAMD64))
	PlatformWasip1Wasm     = Platform(string(GOOSWasip1) + "/" + string(GOARCHWasm))
	PlatformWindows386     = Platform(string(GOOSWindows) + "/" + string(GOARCH386))   // win-32
	PlatformWindowsAMD64   = Platform(string(GOOSWindows) + "/" + string(GOARCHAMD64)) // win-64
	PlatformWindowsARM64   = Platform(string(GOOSWindows) + "/" + string(GOARCHARM64)) // win-arm64
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
	return p.GOOS(), p.GOARCH()
}

// Target converts p to a Target.
func (p Platform) Target() Target {
	return Target{GOOS: p.GOOS().String(), GOARCH: p.GOARCH().String()}
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
