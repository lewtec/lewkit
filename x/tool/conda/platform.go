package conda

import (
	"fmt"
	"runtime"
)

// Subdir is a conda channel platform directory.
//
//lewkit:enum
type Subdir string

const (
	SubdirLinux64      Subdir = "linux-64"      // linux/amd64, linux_amd64
	SubdirLinuxAArch64 Subdir = "linux-aarch64" // linux/arm64, linux_arm64
	SubdirLinux32      Subdir = "linux-32"      // linux/386, linux_386
	SubdirLinuxPPC64le Subdir = "linux-ppc64le" // linux/ppc64le, linux_ppc64le
	SubdirLinuxS390x   Subdir = "linux-s390x"   // linux/s390x, linux_s390x
	SubdirLinuxRiscv64 Subdir = "linux-riscv64" // linux/riscv64, linux_riscv64
	SubdirOSX64        Subdir = "osx-64"        // darwin/amd64, darwin_amd64
	SubdirOSXARM64     Subdir = "osx-arm64"     // darwin/arm64, darwin_arm64
	SubdirWin64        Subdir = "win-64"        // windows/amd64, windows_amd64
	SubdirWinARM64     Subdir = "win-arm64"     // windows/arm64, windows_arm64
	SubdirWin32        Subdir = "win-32"        // windows/386, windows_386
	SubdirNoarch       Subdir = "noarch"
)

const subdirNoarch = "noarch"

func currentSubdir() (string, error) {
	return hostSubdir(runtime.GOOS, runtime.GOARCH)
}

func hostSubdir(goos, goarch string) (string, error) {
	sub, err := ParseSubdir(goos + "/" + goarch)
	if err != nil || sub == SubdirNoarch {
		return "", fmt.Errorf("%w: %s/%s", ErrNoBuild, goos, goarch)
	}
	return string(sub), nil
}
