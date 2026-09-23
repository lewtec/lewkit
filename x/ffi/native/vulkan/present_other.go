//go:build !linux && !windows && !darwin

package vulkan

import "fmt"

func surfaceExtensions() []string { return nil }

func loadHost(*wsi, *Device) error { return nil }

func attachHost(*Screen, int, uintptr, uintptr) (hostSurface, error) {
	return nil, ErrUnavailable
}

func openHost(*Screen, int, int, string) (hostSurface, error) {
	return nil, fmt.Errorf("%w: no vulkan surface", ErrUnavailable)
}
