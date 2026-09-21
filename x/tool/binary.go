package tool

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindBinary returns the first existing candidate for binaryName under baseDirectory.
// Candidates are bin/ and the directory root, each with no extension and .exe, .cmd, .bat.
func FindBinary(baseDirectory, binaryName string) string {
	for _, path := range BinaryCandidates(baseDirectory, binaryName) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// BinaryCandidates lists the paths FindBinary checks, in order.
func BinaryCandidates(baseDirectory, binaryName string) []string {
	extensions := []string{"", ".exe", ".cmd", ".bat"}
	out := make([]string, 0, 2*len(extensions))
	for _, directory := range []string{
		filepath.Join(baseDirectory, "bin"),
		baseDirectory,
	} {
		for _, extension := range extensions {
			out = append(out, filepath.Join(directory, binaryName+extension))
		}
	}
	return out
}

// EnsureBinary runs install, then locates binaryName under destination.
// label is the tool name used in the not-found error.
// fallbackNames are tried in order when binaryName is missing.
func EnsureBinary(destination, binaryName, label string, install func() error, fallbackNames ...string) (string, error) {
	if err := install(); err != nil {
		return "", err
	}
	if path := FindBinary(destination, binaryName); path != "" {
		return path, nil
	}
	for _, name := range fallbackNames {
		if name == "" || name == binaryName {
			continue
		}
		if path := FindBinary(destination, name); path != "" {
			return path, nil
		}
	}
	return "", fmt.Errorf("binary %q not found in %s installation at %s", binaryName, label, destination)
}

// EnsureBinaryNamed is EnsureBinary with an empty binaryName replaced by defaultName.
func EnsureBinaryNamed(destination, binaryName, defaultName, label string, install func() error, fallbackNames ...string) (string, error) {
	if binaryName == "" {
		binaryName = defaultName
	}
	return EnsureBinary(destination, binaryName, label, install, fallbackNames...)
}
