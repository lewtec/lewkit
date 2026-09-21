package tool

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptySpec is returned when Parse is given an empty string.
var ErrEmptySpec = errors.New("spec cannot be empty")

// Spec is a parsed backend:ref@version reference.
// A missing backend is registry. A missing version is latest.
type Spec struct {
	Backend string
	Package string
	Version string
}

// Parse splits a spec string into backend, package, and version.
func Parse(input string) (Spec, error) {
	if input == "" {
		return Spec{}, ErrEmptySpec
	}

	const defaultBackend = "registry"
	const defaultVersion = "latest"

	backend, rest, ok := strings.Cut(input, ":")
	if !ok {
		backend = defaultBackend
		rest = input
	}

	packageRef, versionPart, ok := strings.Cut(rest, "@")
	version := defaultVersion
	if ok {
		version = versionPart
	}

	return Spec{
		Backend: backend,
		Package: packageRef,
		Version: version,
	}, nil
}

// String renders backend:package@version.
func (spec Spec) String() string {
	return fmt.Sprintf("%s:%s@%s", spec.Backend, spec.Package, spec.Version)
}

// Directory is the filesystem key for this backend and package.
func (spec Spec) Directory() string {
	return DirectoryName(spec.Backend, spec.Package)
}

// DirectoryName builds a filesystem-safe directory name from a backend id and package ref.
func DirectoryName(backend, packageRef string) string {
	name := fmt.Sprintf("%s-%s", backend, packageRef)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}
