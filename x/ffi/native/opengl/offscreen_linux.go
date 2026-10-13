//go:build linux

package opengl

// OpenOffscreen draws into a framebuffer the caller can read back.
func OpenOffscreen(width, height int) (*Screen, error) {
	ctx, err := openEGLOffscreen(false)
	if err != nil {
		return nil, err
	}
	return attach(ctx, width, height, true, 0)
}

// OpenDevice opens an offscreen compute context. It does not attach a window.
func OpenDevice() (*Device, error) {
	ctx, err := openEGLOffscreen(true)
	if err != nil {
		return nil, err
	}
	return openCompute(ctx)
}
