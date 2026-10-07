//go:build windows && (amd64 || arm64)

package d3d12

import (
	"log/slog"
	"math"
	"sync"
	"unsafe"
)

// Screen draws one GUI frame into an HWND the caller owns.
type Screen struct {
	mu                        sync.Mutex
	eng                       *engine
	swap                      uintptr
	hwnd                      uintptr
	w, h                      int
	heap                      uintptr
	rtvStart                  uintptr
	rtvInc                    uint32
	back                      [2]uintptr
	rtv                       [2]uintptr
	root                      uintptr
	clearPSO, fillPSO, inkPSO uintptr
	instBuf, underBuf, inkBuf upBuf
	closed                    bool
}

type upBuf struct {
	res, ptr, addr uintptr
	size           int
}

// OpenNative opens a flip-model swapchain for a Win32 HWND. kind 2 is that HWND.
func OpenNative(kind int, hwnd uintptr, width, height int) (s *Screen, err error) {
	if kind != 2 || hwnd == 0 {
		return nil, ErrUnavailable
	}
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		return nil, ErrSize
	}
	s = &Screen{hwnd: hwnd, w: width, h: height}
	defer func() {
		if err != nil {
			_ = s.Close()
			s = nil
		}
	}()
	s.eng, err = newEngine(listDirect, true)
	if err != nil {
		return s, err
	}
	s.root, err = graphicsRoot(s.eng.dev)
	if err != nil {
		return s, err
	}
	if err = s.pipelines(); err != nil {
		return s, err
	}
	s.swap, err = createSwap(s.eng.factory, s.eng.queue, hwnd, width, height)
	if err != nil {
		return s, err
	}
	s.heap, s.rtvStart, s.rtvInc, err = s.eng.rtvHeap()
	if err != nil {
		return s, err
	}
	if err = s.bindBacks(); err != nil {
		return s, err
	}
	slog.Debug("d3d12 screen", "adapter", s.eng.name, "level", s.eng.level)
	return s, nil
}

func (s *Screen) pipelines() error {
	vs, err := compileHLSL(ShaderClear, "clear_vert", "vs_5_0")
	if err != nil {
		return err
	}
	ps, err := compileHLSL(ShaderClear, "clear_frag", "ps_5_0")
	if err != nil {
		return err
	}
	s.clearPSO, err = s.eng.graphicsPSO(s.root, vs, ps, false)
	if err != nil {
		return err
	}
	vs, err = compileHLSL(ShaderFill, "fill_vert", "vs_5_0")
	if err != nil {
		return err
	}
	ps, err = compileHLSL(ShaderFill, "fill_frag", "ps_5_0")
	if err != nil {
		return err
	}
	s.fillPSO, err = s.eng.graphicsPSO(s.root, vs, ps, true)
	if err != nil {
		return err
	}
	vs, err = compileHLSL(ShaderInk, "ink_vert", "vs_5_0")
	if err != nil {
		return err
	}
	ps, err = compileHLSL(ShaderInk, "ink_frag", "ps_5_0")
	if err != nil {
		return err
	}
	s.inkPSO, err = s.eng.graphicsPSO(s.root, vs, ps, true)
	return err
}

func (e *engine) rtvHeap() (heap, start uintptr, inc uint32, err error) {
	hd := heapDesc{typ: heapRTV, num: 2}
	heap, err = newCOM(e.dev, slotDevHeap, &iidHeap, []any{&hd}, uintptr(unsafe.Pointer(&hd)))
	if err != nil {
		return 0, 0, 0, err
	}
	inc = uint32(syscallV(e.dev, slotDevInc, uintptr(heapRTV)))
	start = syscallV(heap, slotHeapCPU)
	if inc == 0 || start == 0 {
		return heap, start, inc, ErrUnavailable
	}
	return heap, start, inc, nil
}

func createSwap(factory, queue, hwnd uintptr, w, h int) (uintptr, error) {
	desc := swapDesc{
		w:           uint32(w),
		h:           uint32(h),
		format:      fmtRGBA8,
		sampleCount: 1,
		usage:       swapUsageRT,
		count:       2,
		scaling:     0,
		effect:      swapEffectFlip,
		alpha:       swapAlphaIgnore,
	}
	var raw uintptr
	var p pins
	p.keep(&desc)
	p.keep(&raw)
	err := callHR(factory, slotSwapChainHwnd,
		queue,
		hwnd,
		uintptr(unsafe.Pointer(&desc)),
		0,
		0,
		uintptr(unsafe.Pointer(&raw)),
	)
	p.done()
	if err != nil || raw == 0 {
		release(raw)
		if err == nil {
			err = ErrUnavailable
		}
		return 0, err
	}
	swap, qerr := query(raw, iidSwap3)
	release(raw)
	if qerr != nil {
		return 0, qerr
	}
	_ = callHR(factory, slotFactoryAssoc, hwnd, noAltEnter)
	return swap, nil
}

func (s *Screen) bindBacks() error {
	s.dropBacks()
	if s.swap == 0 || s.eng == nil {
		return ErrClosed
	}
	for i := 0; i < 2; i++ {
		res, err := newCOM(s.swap, slotSwapBuffer, &iidRes, nil, uintptr(i))
		if err != nil {
			return err
		}
		s.back[i] = res
		handle := s.rtvStart + uintptr(i)*uintptr(s.rtvInc)
		s.rtv[i] = handle
		syscallV(s.eng.dev, slotDevRTV, res, 0, handle)
	}
	return nil
}

func (s *Screen) dropBacks() {
	for i := range s.back {
		release(s.back[i])
		s.back[i] = 0
		s.rtv[i] = 0
	}
}

func (s *Screen) resize(w, h int) error {
	s.dropBacks()
	if err := callHR(s.swap, slotSwapResize, 2, uintptr(w), uintptr(h), uintptr(fmtRGBA8), 0); err != nil {
		return err
	}
	s.w, s.h = w, h
	return s.bindBacks()
}

// Draw paints under, then rounded rects, then glyph ink, and presents.
func (s *Screen) Draw(instances, under, ink []byte, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.eng == nil || s.swap == 0 {
		return ErrClosed
	}
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		return ErrSize
	}
	if len(instances)%64 != 0 {
		return ErrSize
	}
	if width != s.w || height != s.h {
		if err := s.eng.wait(); err != nil {
			return err
		}
		if err := s.resize(width, height); err != nil {
			return err
		}
	}
	if err := s.eng.begin(); err != nil {
		return err
	}
	if err := s.record(instances, under, ink, width, height); err != nil {
		_ = s.eng.closeList()
		return err
	}
	if err := s.eng.submit(); err != nil {
		return err
	}
	return callHR(s.swap, slotSwapPresent, 1, 0)
}

func (s *Screen) record(instances, under, ink []byte, width, height int) error {
	idx := int(syscallV(s.swap, slotSwapIndex))
	if idx != 0 && idx != 1 {
		return ErrLost
	}
	bb := s.back[idx]
	s.eng.transition(bb, statePresent, stateRT)
	s.eng.viewport(width, height)
	s.eng.bindRT(s.rtv[idx])
	syscallV(s.eng.list, slotTopo, uintptr(topoStrip))
	syscallV(s.eng.list, slotGfxRoot, s.root)
	syscallV(s.eng.list, slotSetPSO, s.clearPSO)
	syscallV(s.eng.list, slotDraw, 4, 1, 0, 0)
	need := width * height * 4
	if len(under) >= need {
		if err := s.blit(&s.underBuf, under[:need], width, height); err != nil {
			return err
		}
	}
	if len(instances) >= 64 {
		if err := s.fills(instances, width, height); err != nil {
			return err
		}
	}
	if len(ink) >= need {
		if err := s.blit(&s.inkBuf, ink[:need], width, height); err != nil {
			return err
		}
	}
	s.eng.transition(bb, stateRT, statePresent)
	return nil
}

func (s *Screen) blit(slot *upBuf, data []byte, width, height int) error {
	if err := s.ensure(slot, len(data)); err != nil {
		return err
	}
	if err := slot.write(data); err != nil {
		return err
	}
	syscallV(s.eng.list, slotSetPSO, s.inkPSO)
	syscallV(s.eng.list, slotGfxSRV, 0, slot.addr)
	s.eng.constants(slotGfxConst, 1, []uint32{uint32(width), uint32(height), 0, 0})
	syscallV(s.eng.list, slotDraw, 4, 1, 0, 0)
	return nil
}

func (s *Screen) fills(instances []byte, width, height int) error {
	if err := s.ensure(&s.instBuf, len(instances)); err != nil {
		return err
	}
	if err := s.instBuf.write(instances); err != nil {
		return err
	}
	syscallV(s.eng.list, slotSetPSO, s.fillPSO)
	syscallV(s.eng.list, slotGfxSRV, 0, s.instBuf.addr)
	var c [4]uint32
	c[0] = math.Float32bits(float32(width))
	c[1] = math.Float32bits(float32(height))
	s.eng.constants(slotGfxConst, 1, c[:])
	syscallV(s.eng.list, slotDraw, 4, uintptr(len(instances)/64), 0, 0)
	return nil
}

func (s *Screen) ensure(slot *upBuf, n int) error {
	if slot.res != 0 && slot.size >= n {
		return nil
	}
	s.releaseUp(slot)
	size := n
	if size < 256 {
		size = 256
	}
	size = (size + 255) &^ 255
	buf, err := s.eng.mapUpload(size)
	if err != nil {
		return err
	}
	*slot = buf
	return nil
}

func (e *engine) mapUpload(size int) (upBuf, error) {
	res, err := e.committed(heapUpload, stateGeneric, bufferDesc(size, 0))
	if err != nil {
		return upBuf{}, err
	}
	var ptr uintptr
	empty := cpuRange{}
	var p pins
	p.keep(&empty)
	p.keep(&ptr)
	err = callHR(res, slotResMap, 0, uintptr(unsafe.Pointer(&empty)), uintptr(unsafe.Pointer(&ptr)))
	p.done()
	if err != nil || ptr == 0 {
		release(res)
		if err == nil {
			err = ErrUnavailable
		}
		return upBuf{}, err
	}
	return upBuf{res: res, ptr: ptr, addr: syscallV(res, slotResVA), size: size}, nil
}

func (u *upBuf) write(p []byte) error {
	if u == nil || u.ptr == 0 || len(p) > u.size {
		return ErrSize
	}
	if len(p) == 0 {
		return nil
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(u.ptr)), u.size)
	copy(dst, p)
	return nil
}

func (s *Screen) releaseUp(slot *upBuf) {
	if slot == nil || slot.res == 0 {
		return
	}
	syscallV(slot.res, slotResUnmap, 0, 0)
	release(slot.res)
	*slot = upBuf{}
}

// Adopt points the swapchain at a replacement HWND, or resizes it.
func (s *Screen) Adopt(hwnd uintptr, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.eng == nil {
		return ErrClosed
	}
	if hwnd == 0 {
		return ErrLost
	}
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		return ErrSize
	}
	if err := s.eng.wait(); err != nil {
		return err
	}
	if hwnd != s.hwnd {
		s.dropBacks()
		release(s.swap)
		s.swap = 0
		swap, err := createSwap(s.eng.factory, s.eng.queue, hwnd, width, height)
		if err != nil {
			return err
		}
		s.swap = swap
		s.hwnd = hwnd
		s.w, s.h = width, height
		return s.bindBacks()
	}
	if width != s.w || height != s.h {
		return s.resize(width, height)
	}
	return nil
}

// Close releases the swapchain and the device.
func (s *Screen) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.eng != nil {
		_ = s.eng.wait()
	}
	s.dropBacks()
	release(s.swap)
	s.swap = 0
	s.releaseUp(&s.instBuf)
	s.releaseUp(&s.underBuf)
	s.releaseUp(&s.inkBuf)
	release(s.clearPSO)
	release(s.fillPSO)
	release(s.inkPSO)
	release(s.root)
	release(s.heap)
	s.clearPSO, s.fillPSO, s.inkPSO = 0, 0, 0
	s.root, s.heap = 0, 0
	if s.eng != nil {
		s.eng.release()
		s.eng = nil
	}
	return nil
}
