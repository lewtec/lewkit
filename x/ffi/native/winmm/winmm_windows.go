//go:build windows

package winmm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	waveMapper = 0xFFFFFFFF
	whdrDone   = 1
)

var (
	errClosed = errors.New("winmm stream closed")
	errDevice = errors.New("winmm device")
	errWinmm  = errors.New("winmm")
)

var (
	loadOnce sync.Once
	loadErr  error

	waveOutGetNumDevs    func() uint32
	waveOutGetDevCapsW   func(id uintptr, caps uintptr, size uint32) uint32
	waveOutOpen          func(out *uintptr, id uint32, format uintptr, callback, instance uintptr, flags uint32) uint32
	waveOutPrepareHeader func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutUnprepare     func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutWrite         func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutReset         func(h uintptr) uint32
	waveOutClose         func(h uintptr) uint32
)

// Available loads winmm.dll.
func Available() error {
	if !layoutOK() {
		return errUnavailable
	}
	loadOnce.Do(func() { loadErr = bindAll() })
	return loadErr
}

func bindAll() error {
	lib, err := native.Open("winmm.dll", native.Lazy)
	if err != nil {
		return err
	}
	binds := []struct {
		name string
		fn   any
	}{
		{"waveOutGetNumDevs", &waveOutGetNumDevs},
		{"waveOutGetDevCapsW", &waveOutGetDevCapsW},
		{"waveOutOpen", &waveOutOpen},
		{"waveOutPrepareHeader", &waveOutPrepareHeader},
		{"waveOutUnprepareHeader", &waveOutUnprepare},
		{"waveOutWrite", &waveOutWrite},
		{"waveOutReset", &waveOutReset},
		{"waveOutClose", &waveOutClose},
	}
	for _, item := range binds {
		if _, err := native.Symbol(lib, item.name); err != nil {
			return fmt.Errorf("%s: %w", item.name, err)
		}
		native.Func(lib, item.name, item.fn)
	}
	return nil
}

func statusErr(code uint32) error {
	if code == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d", errWinmm, code)
}

// Devices lists waveOut endpoints.
func Devices() ([]Device, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	n := int(waveOutGetNumDevs())
	out := make([]Device, 0, n)
	for i := 0; i < n; i++ {
		var caps waveCaps
		rc := waveOutGetDevCapsW(uintptr(i), uintptr(unsafe.Pointer(&caps)), uint32(unsafe.Sizeof(caps)))
		if rc != 0 {
			return nil, statusErr(rc)
		}
		out = append(out, Device{ID: strconv.Itoa(i), Name: utf16z(caps.name[:])})
	}
	return out, nil
}

// Stream plays PCM through one waveOut handle.
type Stream struct {
	mu       sync.Mutex
	ctx      context.Context
	h        uintptr
	hdr      waveHdr
	buf      []byte
	prepared bool
	closed   bool
}

// Open plays layout on id. An empty id uses WAVE_MAPPER.
// Write waits out each buffer on the device clock. A cancelled ctx resets it.
func Open(ctx context.Context, id string, layout Layout) (*Stream, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	device := uint32(waveMapper)
	if id != "" {
		n, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errDevice, id)
		}
		device = uint32(n)
	}
	bits := uint16(16)
	block := layout.Channels * 2
	if layout.Sample == SampleF32LE {
		bits = 32
		block = layout.Channels * 4
	}
	format := waveFormat{
		tag:      uint16(layout.Sample),
		channels: uint16(layout.Channels),
		rate:     uint32(layout.Rate),
		avg:      uint32(layout.Rate * block),
		block:    uint16(block),
		bits:     bits,
	}
	var h uintptr
	rc := waveOutOpen(&h, device, uintptr(unsafe.Pointer(&format)), 0, 0, 0)
	if rc != 0 {
		return nil, statusErr(rc)
	}
	return &Stream{ctx: ctx, h: h}, nil
}

// Write blocks until the device finishes p.
func (s *Stream) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.h == 0 {
		return errClosed
	}
	if err := s.unprepare(); err != nil {
		return err
	}
	s.buf = append([]byte(nil), p...)
	s.hdr = waveHdr{
		data:   uintptr(unsafe.Pointer(&s.buf[0])),
		length: uint32(len(s.buf)),
	}
	size := uint32(unsafe.Sizeof(s.hdr))
	if rc := waveOutPrepareHeader(s.h, uintptr(unsafe.Pointer(&s.hdr)), size); rc != 0 {
		return statusErr(rc)
	}
	s.prepared = true
	if rc := waveOutWrite(s.h, uintptr(unsafe.Pointer(&s.hdr)), size); rc != 0 {
		return statusErr(rc)
	}
	for s.hdr.flags&whdrDone == 0 {
		if s.closed {
			return errClosed
		}
		if err := s.ctx.Err(); err != nil {
			waveOutReset(s.h)
			return err
		}
		s.mu.Unlock()
		time.Sleep(time.Millisecond)
		s.mu.Lock()
	}
	return nil
}

func (s *Stream) unprepare() error {
	if !s.prepared {
		return nil
	}
	rc := waveOutUnprepare(s.h, uintptr(unsafe.Pointer(&s.hdr)), uint32(unsafe.Sizeof(s.hdr)))
	s.prepared = false
	return statusErr(rc)
}

// Close resets the device and closes the handle.
func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.h == 0 {
		return nil
	}
	reset := waveOutReset(s.h)
	prep := s.unprepare()
	rc := waveOutClose(s.h)
	s.h = 0
	err := statusErr(rc)
	if reset != 0 {
		err = statusErr(reset)
	} else if prep != nil {
		err = prep
	}
	return err
}
