//go:build android

package vulkan

import "fmt"

const extHostSurface = "VK_KHR_android_surface"

func surfaceExtensions() []string { return []string{extHostSurface} }

func loadHost(*wsi, *Device) error { return nil }

type androidHost struct {
	screen   *Screen
	window   uintptr
	borrowed bool
	createFn func(inst uintptr, info *androidSurfaceInfo, alloc uintptr, surface *uint64) int32
}

type androidSurfaceInfo struct {
	sType  int32
	pNext  uintptr
	flags  uint32
	window uintptr
}

func attachHost(s *Screen, kind int, a, _ uintptr) (hostSurface, error) {
	if kind != 4 || a == 0 {
		return nil, fmt.Errorf("%w: android window", ErrUnavailable)
	}
	return &androidHost{screen: s, window: a, borrowed: true}, nil
}

func (h *androidHost) create(d *Device, w *wsi) (uint64, error) {
	if err := d.api.bind(d.api.getInstanceProcAddr, d.inst, "vkCreateAndroidSurfaceKHR", &h.createFn); err != nil {
		return 0, err
	}
	_ = w
	info := androidSurfaceInfo{sType: 1000008000, window: h.window}
	var surface uint64
	if err := check(h.createFn(d.inst, &info, 0, &surface)); err != nil {
		return 0, fmt.Errorf("android surface: %w", err)
	}
	return surface, nil
}

func (h *androidHost) poll()    {}
func (h *androidHost) destroy() {}

func init() { adoptNative = adoptAndroid }

func adoptAndroid(s *Screen, window uintptr, width, height int) error {
	if s == nil || s.d == nil {
		return ErrClosed
	}
	if window == 0 {
		return ErrLost
	}
	host, ok := s.host.(*androidHost)
	if !ok || host == nil {
		return s.Fit(width, height)
	}
	if host.window == window && s.swap != 0 && s.width == width && s.height == height {
		return nil
	}
	if err := s.d.WaitIdle(); err != nil {
		return err
	}
	if host.window != window || s.surface == 0 {
		s.dropSwap()
		if s.surface != 0 && s.wsi.destroySurface != nil {
			s.wsi.destroySurface(s.d.inst, s.surface, 0)
			s.surface = 0
		}
		host.window = window
		surface, err := host.create(s.d, &s.wsi)
		if err != nil {
			return err
		}
		s.surface = surface
		s.width, s.height = width, height
		return s.makeSwapchain()
	}
	return s.resizeTo(width, height)
}
