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
type Platform struct {
	GOOS   GOOS
	GOARCH GOARCH
}

var (
	PlatformAixPPC64       = Platform{GOOSAix, GOARCHPPC64}
	PlatformAndroid386     = Platform{GOOSAndroid, GOARCH386}
	PlatformAndroidAMD64   = Platform{GOOSAndroid, GOARCHAMD64}
	PlatformAndroidARM     = Platform{GOOSAndroid, GOARCHARM}
	PlatformAndroidARM64   = Platform{GOOSAndroid, GOARCHARM64}
	PlatformDarwinAMD64    = Platform{GOOSDarwin, GOARCHAMD64} // osx-64
	PlatformDarwinARM64    = Platform{GOOSDarwin, GOARCHARM64} // osx-arm64
	PlatformDragonflyAMD64 = Platform{GOOSDragonfly, GOARCHAMD64}
	PlatformFreebsd386     = Platform{GOOSFreebsd, GOARCH386}
	PlatformFreebsdAMD64   = Platform{GOOSFreebsd, GOARCHAMD64}
	PlatformFreebsdARM     = Platform{GOOSFreebsd, GOARCHARM}
	PlatformFreebsdARM64   = Platform{GOOSFreebsd, GOARCHARM64}
	PlatformFreebsdRiscv64 = Platform{GOOSFreebsd, GOARCHRiscv64}
	PlatformIllumosAMD64   = Platform{GOOSIllumos, GOARCHAMD64}
	PlatformIOSAMD64       = Platform{GOOSIOS, GOARCHAMD64}
	PlatformIOSARM64       = Platform{GOOSIOS, GOARCHARM64}
	PlatformJSWasm         = Platform{GOOSJS, GOARCHWasm}
	PlatformLinux386       = Platform{GOOSLinux, GOARCH386}   // linux-32
	PlatformLinuxAMD64     = Platform{GOOSLinux, GOARCHAMD64} // linux-64
	PlatformLinuxARM       = Platform{GOOSLinux, GOARCHARM}
	PlatformLinuxARM64     = Platform{GOOSLinux, GOARCHARM64} // linux-aarch64
	PlatformLinuxLoong64   = Platform{GOOSLinux, GOARCHLoong64}
	PlatformLinuxMips      = Platform{GOOSLinux, GOARCHMips}
	PlatformLinuxMips64    = Platform{GOOSLinux, GOARCHMips64}
	PlatformLinuxMips64le  = Platform{GOOSLinux, GOARCHMips64le}
	PlatformLinuxMipsle    = Platform{GOOSLinux, GOARCHMipsle}
	PlatformLinuxPPC64     = Platform{GOOSLinux, GOARCHPPC64}
	PlatformLinuxPPC64le   = Platform{GOOSLinux, GOARCHPPC64le} // linux-ppc64le
	PlatformLinuxRiscv64   = Platform{GOOSLinux, GOARCHRiscv64} // linux-riscv64
	PlatformLinuxS390x     = Platform{GOOSLinux, GOARCHS390x}   // linux-s390x
	PlatformLinuxSparc64   = Platform{GOOSLinux, GOARCHSparc64}
	PlatformLinuxISOAMD64  = Platform{GOOSLinuxISO, GOARCHAMD64}
	PlatformLinuxISOARM64  = Platform{GOOSLinuxISO, GOARCHARM64}
	PlatformLinuxUKIAMD64  = Platform{GOOSLinuxUKI, GOARCHAMD64}
	PlatformLinuxUKIARM64  = Platform{GOOSLinuxUKI, GOARCHARM64}
	PlatformMBRAMD64       = Platform{GOOSMBR, GOARCHAMD64}
	PlatformNetbsd386      = Platform{GOOSNetbsd, GOARCH386}
	PlatformNetbsdAMD64    = Platform{GOOSNetbsd, GOARCHAMD64}
	PlatformNetbsdARM      = Platform{GOOSNetbsd, GOARCHARM}
	PlatformNetbsdARM64    = Platform{GOOSNetbsd, GOARCHARM64}
	PlatformOpenbsd386     = Platform{GOOSOpenbsd, GOARCH386}
	PlatformOpenbsdAMD64   = Platform{GOOSOpenbsd, GOARCHAMD64}
	PlatformOpenbsdARM     = Platform{GOOSOpenbsd, GOARCHARM}
	PlatformOpenbsdARM64   = Platform{GOOSOpenbsd, GOARCHARM64}
	PlatformOpenbsdPPC64   = Platform{GOOSOpenbsd, GOARCHPPC64}
	PlatformOpenbsdRiscv64 = Platform{GOOSOpenbsd, GOARCHRiscv64}
	PlatformPlan9386       = Platform{GOOSPlan9, GOARCH386}
	PlatformPlan9AMD64     = Platform{GOOSPlan9, GOARCHAMD64}
	PlatformPlan9ARM       = Platform{GOOSPlan9, GOARCHARM}
	PlatformSolarisAMD64   = Platform{GOOSSolaris, GOARCHAMD64}
	PlatformTamagoAMD64    = Platform{GOOSTamago, GOARCHAMD64}
	PlatformTamagoARM      = Platform{GOOSTamago, GOARCHARM}
	PlatformTamagoARM64    = Platform{GOOSTamago, GOARCHARM64}
	PlatformTamagoRiscv64  = Platform{GOOSTamago, GOARCHRiscv64}
	PlatformUEFIAMD64      = Platform{GOOSUEFI, GOARCHAMD64}
	PlatformWasip1Wasm     = Platform{GOOSWasip1, GOARCHWasm}
	PlatformWindows386     = Platform{GOOSWindows, GOARCH386}   // win-32
	PlatformWindowsAMD64   = Platform{GOOSWindows, GOARCHAMD64} // win-64
	PlatformWindowsARM64   = Platform{GOOSWindows, GOARCHARM64} // win-arm64
)

// HostPlatform returns the process Platform.
func HostPlatform() Platform {
	return Platform{HostGOOS(), HostGOARCH()}
}

// PlatformFor resolves the Platform for goos and goarch.
func PlatformFor(goos GOOS, goarch GOARCH) (Platform, error) {
	return ParsePlatform(string(goos) + "/" + string(goarch))
}

// Split returns the GOOS and GOARCH components of p.
func (p Platform) Split() (GOOS, GOARCH) {
	return p.GOOS, p.GOARCH
}

// Target converts p to a Target.
func (p Platform) Target() Target {
	return Target{GOOS: p.GOOS.String(), GOARCH: p.GOARCH.String()}
}

// AndroidABI returns the Android ABI for p.
func (p Platform) AndroidABI() (ABI, error) {
	if p.GOOS != GOOSAndroid {
		return "", fmt.Errorf("%w: %s", ErrABI, p)
	}
	return ABIFor(p.GOARCH)
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
