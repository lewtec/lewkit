//go:build !android && !linux && !windows && !darwin

package vulkan

func surfaceExtensions() []string { return nil }

func loadHost(*wsi, *Device) error { return nil }

func attachHost(*Screen, int, uintptr, uintptr) (hostSurface, error) {
	return nil, ErrUnavailable
}
