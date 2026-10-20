package build

import "strings"

// Target is one CGO-free desktop binary.
type Target struct {
	GOOS   string
	GOARCH string
}

// Platform resolves t into the uber Platform enum.
func (t Target) Platform() (Platform, error) {
	return PlatformFor(GOOS(t.GOOS), GOARCH(t.GOARCH))
}

// String formats t as goos/goarch.
func (t Target) String() string {
	return t.GOOS + "/" + t.GOARCH
}

// DesktopTargets is the release matrix: Linux, macOS, and Windows on amd64 and arm64.
func DesktopTargets() []Target {
	platforms := DesktopPlatforms()
	out := make([]Target, 0, len(platforms))
	for _, platform := range platforms {
		out = append(out, platform.Target())
	}
	return out
}

// ArchiveName is project_goos_goarch.tar.gz, or .zip when GOOS is windows.
// uefi and linux_uki are .efi. linux_iso is .iso. mbr is .img.
// GOOS and GOARCH are the go env values, so a dist directory can be
// uploaded as release assets without renaming.
func ArchiveName(project string, target Target) string {
	ext := ".tar.gz"
	switch GOOS(target.GOOS) {
	case GOOSWindows:
		ext = ".zip"
	case GOOSUEFI, GOOSLinuxUKI:
		ext = ".efi"
	case GOOSLinuxISO:
		ext = ".iso"
	case GOOSMBR:
		ext = ".img"
	}
	return project + "_" + target.GOOS + "_" + target.GOARCH + ext
}

// AppFile is one host artifact: stem_goos_goarch plus the host suffix.
// Windows is .exe, Android is .apk, Linux is .AppImage, and every other host is .app.
// Android's stem is the last package-id label. Every other host uses product.
func AppFile(product, packageID, goos, goarch string) string {
	stem := product
	ext := ".app"
	switch GOOS(goos) {
	case GOOSAndroid:
		stem = androidLabel(packageID)
		ext = ".apk"
	case GOOSWindows:
		ext = ".exe"
	case GOOSLinux:
		ext = ".AppImage"
	}
	return stem + "_" + goos + "_" + goarch + ext
}

func androidLabel(packageID string) string {
	label := packageID
	if i := strings.LastIndex(label, "."); i >= 0 {
		label = label[i+1:]
	}
	if label == "" {
		return "app"
	}
	return label
}
