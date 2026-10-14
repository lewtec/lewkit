package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// FindBinary returns the first existing candidate for binaryName under baseDirectory.
// An app bundle is first: Name.app/Contents/MacOS/Name, one directory down as well,
// with a case-insensitive match on the final component.
// Then bin/ and the directory root, each with no extension and .exe, .cmd, .bat.
// A disk image is skipped, so a copied .dmg is not executed.
// A symlink counts even when its target sits outside baseDirectory.
// The result is an OS path. An unreadable base directory returns an empty string.
func FindBinary(baseDirectory, binaryName string) string {
	if found := findAppBundleBinary(baseDirectory, binaryName); found != "" {
		return found
	}
	root, err := lewpath.Open(baseDirectory)
	if err != nil {
		return ""
	}
	defer root.Close()
	for _, candidate := range binaryCandidateNames(binaryName) {
		info, err := candidate.Lstat(root)
		if err != nil || info.IsDir() {
			continue
		}
		host := joinHost(root.Name(), candidate)
		if isDiskImageFile(host) {
			continue
		}
		return host
	}
	return ""
}

// findAppBundleBinary looks for binaryName inside a macOS app bundle.
// The bundle may sit at the install root or one directory down (a disk-image volume wrapper).
func findAppBundleBinary(baseDirectory, binaryName string) string {
	if found := appBundleBinaryIn(baseDirectory, binaryName); found != "" {
		return found
	}
	entries, err := os.ReadDir(baseDirectory)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() || isAppBundleName(entry.Name()) {
			continue
		}
		if found := appBundleBinaryIn(filepath.Join(baseDirectory, entry.Name()), binaryName); found != "" {
			return found
		}
	}
	return ""
}

func appBundleBinaryIn(directory, binaryName string) string {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return ""
	}
	var fallback string
	for _, entry := range entries {
		if !entry.IsDir() || !isAppBundleName(entry.Name()) {
			continue
		}
		host := bundleExecutable(filepath.Join(directory, entry.Name()), binaryName)
		if host == "" {
			continue
		}
		if strings.EqualFold(appBundleStem(entry.Name()), binaryName) {
			return host
		}
		if fallback == "" {
			fallback = host
		}
	}
	return fallback
}

func bundleExecutable(appDirectory, binaryName string) string {
	entries, err := os.ReadDir(filepath.Join(appDirectory, "Contents", "MacOS"))
	if err != nil {
		return ""
	}
	var fallback string
	for _, entry := range entries {
		if entry.IsDir() || !executableNameMatches(entry.Name(), binaryName) {
			continue
		}
		host := filepath.Join(appDirectory, "Contents", "MacOS", entry.Name())
		info, err := os.Lstat(host)
		if err != nil || info.IsDir() || isDiskImageFile(host) {
			continue
		}
		if entry.Name() == binaryName {
			return host
		}
		if fallback == "" {
			fallback = host
		}
	}
	return fallback
}

func executableNameMatches(fileName, binaryName string) bool {
	if strings.EqualFold(fileName, binaryName) {
		return true
	}
	for _, extension := range []string{".exe", ".cmd", ".bat"} {
		if strings.EqualFold(fileName, binaryName+extension) {
			return true
		}
	}
	return false
}

// BinaryCandidates lists the OS paths FindBinary checks, in order.
func BinaryCandidates(baseDirectory, binaryName string) []string {
	names := binaryCandidateNames(binaryName)
	out := make([]string, 0, len(names))
	for _, candidate := range names {
		out = append(out, joinHost(baseDirectory, candidate))
	}
	return out
}

func binaryCandidateNames(binaryName string) []lewpath.Path {
	extensions := []string{"", ".exe", ".cmd", ".bat"}
	out := make([]lewpath.Path, 0, 2*len(extensions))
	for _, directory := range []lewpath.Path{lewpath.New("bin"), lewpath.New(".")} {
		for _, extension := range extensions {
			out = append(out, directory.Join(binaryName+extension))
		}
	}
	return out
}

func joinHost(directory string, name lewpath.Path) string {
	if name.String() == "." || name.String() == "" {
		return directory
	}
	return filepath.Join(directory, filepath.FromSlash(name.String()))
}

// EnsureBinary runs install, then locates binaryName under destination.
// label is the tool name used in the not-found error.
// fallbackNames are tried in order when binaryName is missing.
func EnsureBinary(destination, binaryName, label string, install func() error, fallbackNames ...string) (string, error) {
	if err := install(); err != nil {
		return "", err
	}
	if binaryPath := FindBinary(destination, binaryName); binaryPath != "" {
		return binaryPath, nil
	}
	for _, name := range fallbackNames {
		if name == "" || name == binaryName {
			continue
		}
		if binaryPath := FindBinary(destination, name); binaryPath != "" {
			return binaryPath, nil
		}
	}
	return "", fmt.Errorf("binary %q not found in %s %w %s", binaryName, label, ErrInstallationAt, destination)
}

// EnsureBinaryNamed is EnsureBinary with an empty binaryName replaced by defaultName.
func EnsureBinaryNamed(destination, binaryName, defaultName, label string, install func() error, fallbackNames ...string) (string, error) {
	if binaryName == "" {
		binaryName = defaultName
	}
	return EnsureBinary(destination, binaryName, label, install, fallbackNames...)
}
