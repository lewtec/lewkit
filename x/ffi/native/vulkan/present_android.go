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
		behind, err := s.extentBehind(width, height)
		if err != nil {
			return err
		}
		if behind {
			return ErrHold
		}
		if !s.stale && s.width == width && s.height == height {
			return nil
		}
	}
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
		return s.holdIfBehind(width, height)
	}
	if err := s.resizeTo(width, height); err != nil {
		return err
	}
	return s.holdIfBehind(width, height)
}

// extentBehind reports that the view's aspect and the current extent's
// aspect disagree. A swapchain whose turn no longer matches the hint is
// marked stale so the next agreeing frame rebuilds before present.
func (s *Screen) extentBehind(width, height int) (bool, error) {
	if s == nil || s.d == nil || s.surface == 0 || s.wsi.surfaceCaps == nil {
		return false, nil
	}
	var caps surfaceCaps
	if err := check(s.wsi.surfaceCaps(s.d.phys, s.surface, &caps)); err != nil {
		return false, fmt.Errorf("surface caps: %w", err)
	}
	if holdMisaligned(width, height, caps.currentW, caps.currentH, true) {
		return true, nil
	}
	if s.swap != 0 && s.turn != contentTurn(caps) {
		s.stale = true
	}
	return false, nil
}

func (s *Screen) holdIfBehind(width, height int) error {
	behind, err := s.extentBehind(width, height)
	if err != nil || !behind {
		return err
	}
	// This image was built while the extent still had the other aspect.
	// Presenting it later, unchanged, would sit off the window.
	if ow, oh := s.Oriented(); ow != width || oh != height {
		s.stale = true
	}
	return ErrHold
}
