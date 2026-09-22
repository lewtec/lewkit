package tool

import (
	"fmt"
	"path/filepath"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// FindBinary returns the first existing candidate for binaryName under baseDirectory.
// Candidates are bin/ and the directory root, each with no extension and .exe, .cmd, .bat.
// A symlink counts even when its target sits outside baseDirectory.
// The result is an OS path. An unreadable base directory returns an empty string.
func FindBinary(baseDirectory, binaryName string) string {
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
		return joinHost(root.Name(), candidate)
	}
	return ""
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
	return "", fmt.Errorf("binary %q not found in %s installation at %s", binaryName, label, destination)
}

// EnsureBinaryNamed is EnsureBinary with an empty binaryName replaced by defaultName.
func EnsureBinaryNamed(destination, binaryName, defaultName, label string, install func() error, fallbackNames ...string) (string, error) {
	if binaryName == "" {
		binaryName = defaultName
	}
	return EnsureBinary(destination, binaryName, label, install, fallbackNames...)
}
