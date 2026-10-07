//go:build linux && !android

package opengl

import "github.com/lewtec/lewkit/x/ffi/native"

// kindX11 is window.SurfaceX11: Display* in a, Window in b.
const kindX11 = 1

// OpenNative attaches a screen to an X11 window.
func OpenNative(kind int, a, b uintptr, width, height int) (*Screen, error) {
	if kind != kindX11 || a == 0 || b == 0 {
		return nil, ErrUnavailable
	}
	ctx, err := openEGL(a, b, false, false, false)
	if err != nil {
		ctx, err = openEGL(a, b, true, false, false)
	}
	if err != nil {
		var glx glContext
		glx, err = openGLX(a, b)
		if err != nil {
			return nil, err
		}
		return attach(glx, width, height, false, a)
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

// OpenDevice opens a compute context. It does not attach a window.
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
		if err := screen.Draw(nil, nil, nil, 2, 2); err != nil {
			return err
		}
		pix, err := screen.Read()
		if err != nil {
			return err
		}
		if len(pix) < 4 || pix[0] != 0 || pix[1] != 0 || pix[2] != 0 || pix[3] != 255 {
			return ErrUnavailable
		}
		return nil
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
