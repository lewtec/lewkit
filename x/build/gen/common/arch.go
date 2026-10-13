package common

import (
	"fmt"
	"runtime"
)

// Arch accepts amd64 and arm64. An empty name is the process GOARCH.
// Any other name is fmt.Errorf("%w: %s", unsupported, name).
func Arch(goarch string, unsupported error) (string, error) {
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	switch goarch {
	case "amd64", "arm64":
		return goarch, nil
	default:
		return "", fmt.Errorf("%w: %s", unsupported, goarch)
	}
}
