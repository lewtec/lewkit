//go:build android

package vulkan

import (
	"fmt"
)

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
	if err := s.d.live(); err != nil {
		return err
	}
	if window == 0 {
		s.releaseSurface()
		return ErrLost
	}
	host, ok := s.host.(*androidHost)
	if !ok || host == nil {
		return s.Fit(width, height)
	}
	if host.window == window && s.surface != 0 && !s.lost && s.swap != 0 {
		// A matching frame does not read the extent. That read overlapped
		// the previous present and faulted the driver.
		if !extentReadNeeded(true, s.stale, s.width, s.height, width, height) {
			return nil
		}
	}
	// The present fence is the previous picture. Rebuilding reads the same
	// native window, so wait it out first. The extent keeps the old aspect
	// until a new buffer is submitted, so holding here never presents and
	// the compositor scales the old picture into the new window.
	_ = s.d.WaitIdle()
	if host.window != window || s.surface == 0 || s.lost {
		s.dropSwap()
		if s.surface != 0 && s.wsi.destroySurface != nil {
			s.wsi.destroySurface(s.d.inst, s.surface, 0)
			s.surface = 0
		}
		s.lost = false
		host.window = window
		surface, err := host.create(s.d, &s.wsi)
		if err != nil {
			s.stale = true
			if surfaceGone(err) {
				return ErrLost
			}
			return err
		}
		s.surface = surface
		s.holdSize(width, height)
		if err := s.makeSwapchain(); err != nil {
			if surfaceGone(err) {
				s.stale = true
				if surfaceDead(err) {
					s.lost = true
				}
				return ErrLost
			}
			return err
		}
		s.stale = false
		return nil
	}
	return s.resizeTo(width, height)
}
