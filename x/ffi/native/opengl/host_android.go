//go:build android

package opengl

import "github.com/lewtec/lewkit/x/ffi/native"

// kindAndroid is window.SurfaceAndroid: an ANativeWindow* in a.
const kindAndroid = 4

// OpenNative attaches an OpenGL ES screen to an Android window.
func OpenNative(kind int, a, _ uintptr, width, height int) (*Screen, error) {
	if kind != kindAndroid || a == 0 {
		return nil, ErrUnavailable
	}
	ctx, err := openEGL(0, a, true, false, true)
	if err != nil {
		return nil, err
	}
	return attach(ctx, width, height, false, a)
}

// OpenOffscreen draws into a framebuffer the caller can read back.
func OpenOffscreen(width, height int) (*Screen, error) {
	ctx, err := openEGLOffscreen(false)
	if err != nil {
		return nil, err
	}
	return attach(ctx, width, height, true, 0)
}

// OpenDevice opens an OpenGL ES 3.1 compute context.
func OpenDevice() (*Device, error) {
	ctx, err := openEGLOffscreen(true)
	if err != nil {
		return nil, err
	}
	return openCompute(ctx)
}

var (
	presentProbe = native.Once(func() error {
		screen, err := OpenOffscreen(2, 2)
		if err != nil {
			return err
		}
		defer screen.Close()
		return screen.Draw(nil, nil, nil, 2, 2)
	})
	computeProbe = native.Once(func() error {
		device, err := OpenDevice()
		if err != nil {
			return err
		}
		defer device.Close()
		return probeCompute(device)
	})
)

// Available reports whether a present context can be created.
func Available() error { return presentProbe() }

// ComputeAvailable reports whether a compute context can dispatch.
func ComputeAvailable() error { return computeProbe() }
