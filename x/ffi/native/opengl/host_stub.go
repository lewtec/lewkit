//go:build !linux && !windows && !darwin && !ios

package opengl

// OpenNative reports that this OS has no OpenGL screen.
func OpenNative(int, uintptr, uintptr, int, int) (*Screen, error) {
	return nil, ErrUnavailable
}

// OpenOffscreen reports that this OS has no OpenGL screen.
func OpenOffscreen(int, int) (*Screen, error) { return nil, ErrUnavailable }

// OpenDevice reports that this OS has no OpenGL compute device.
func OpenDevice() (*Device, error) { return nil, ErrUnavailable }

// Available reports that this OS has no OpenGL screen.
func Available() error { return ErrUnavailable }

// ComputeAvailable reports that this OS has no OpenGL compute device.
func ComputeAvailable() error { return ErrUnavailable }
