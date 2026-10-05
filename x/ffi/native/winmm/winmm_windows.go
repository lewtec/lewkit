//go:build windows

package winmm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	waveMapper    = 0xFFFFFFFF
	whdrDone      = 1
	callbackEvent = 0x00050000
	waitTimeout   = 0x102
)

var (
	errClosed = errors.New("winmm stream closed")
	errDevice = errors.New("winmm device")
	errWinmm  = errors.New("winmm")
)

var (
	waveOutGetNumDevs    func() uint32
	waveOutGetDevCapsW   func(id uintptr, caps uintptr, size uint32) uint32
	waveOutOpen          func(out *uintptr, id uint32, format uintptr, callback, instance uintptr, flags uint32) uint32
	waveOutPrepareHeader func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutUnprepare     func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutWrite         func(h uintptr, hdr uintptr, size uint32) uint32
	waveOutReset         func(h uintptr) uint32
	waveOutClose         func(h uintptr) uint32

	procCreateEvent = native.ProcOf("kernel32.dll", "CreateEventW")
	procSetEvent    = native.ProcOf("kernel32.dll", "SetEvent")
	procResetEvent  = native.ProcOf("kernel32.dll", "ResetEvent")
	procWait        = native.ProcOf("kernel32.dll", "WaitForSingleObject")
	procCloseHandle = native.ProcOf("kernel32.dll", "CloseHandle")
)

var available = native.Once(bindAll)

// Available loads winmm.dll.
func Available() error {
	if !layoutOK() {
		return errUnavailable
	}
	return available()
}

func bindAll() error {
	lib, err := native.OpenChain(native.Lazy, "winmm.dll")
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
		if err := native.Bind(lib, item.name, item.fn); err != nil {
			return err
		}
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
	event    uintptr
	hdr      waveHdr
	buf      []byte
	chunk    int
	block    int
	prepared bool
	closed   bool
	busy     bool
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
	event, _, _ := procCreateEvent.Call(0, 1, 0, 0)
	if event == 0 {
		return nil, fmt.Errorf("%w: event", errWinmm)
	}
	var h uintptr
	rc := waveOutOpen(&h, device, uintptr(unsafe.Pointer(&format)), event, 0, callbackEvent)
	if rc != 0 {
		procCloseHandle.Call(event)
		return nil, statusErr(rc)
	}
	return &Stream{
		ctx:   ctx,
		h:     h,
		event: event,
		chunk: chunkBytes(layout.Rate, block),
		block: block,
	}, nil
}

// Write blocks until the device finishes p.
// PCM goes out in short buffers. The device signals an event when each one finishes.
func (s *Stream) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	s.mu.Lock()
	s.busy = true
	defer func() {
		s.busy = false
		if s.closed {
			s.shutdown()
		}
		s.mu.Unlock()
	}()
	if s.closed || s.h == 0 {
		return errClosed
	}
	for len(p) > 0 {
		if s.closed {
			return errClosed
		}
		if err := s.ctx.Err(); err != nil {
			if s.h != 0 {
				waveOutReset(s.h)
			}
			return err
		}
		n := s.chunk
		if n > len(p) {
			n = len(p)
		} else if s.block > 1 {
			n -= n % s.block
			if n == 0 {
				n = len(p)
			}
		}
		if err := s.play(p[:n]); err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}

func (s *Stream) play(p []byte) error {
	if err := s.unprepare(); err != nil {
		return err
	}
	if s.event != 0 {
		procResetEvent.Call(s.event)
	}
	s.buf = append(s.buf[:0], p...)
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
	return s.waitDone()
}

func (s *Stream) waitDone() error {
	for {
		if s.closed {
			return errClosed
		}
		if err := s.ctx.Err(); err != nil {
			if s.h != 0 {
				waveOutReset(s.h)
			}
			return err
		}
		event := s.event
		s.mu.Unlock()
		rc, _, _ := procWait.Call(event, 50)
		s.mu.Lock()
		if s.closed {
			return errClosed
		}
		if err := s.ctx.Err(); err != nil {
			if s.h != 0 {
				waveOutReset(s.h)
			}
			return err
		}
		if rc == 0 || s.hdr.flags&whdrDone != 0 {
			return nil
		}
		if rc != waitTimeout {
			return fmt.Errorf("%w: wait %d", errWinmm, rc)
		}
	}
}

func (s *Stream) unprepare() error {
	if !s.prepared {
		return nil
	}
	rc := waveOutUnprepare(s.h, uintptr(unsafe.Pointer(&s.hdr)), uint32(unsafe.Sizeof(s.hdr)))
	s.prepared = false
	return statusErr(rc)
}

// Close stops playback and closes the handle.
// A Write that is waiting drains the current buffer after Reset marks it done.
func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed && !s.busy {
		return nil
	}
	s.closed = true
	var reset uint32
	if s.h != 0 {
		reset = waveOutReset(s.h)
	}
	if s.event != 0 {
		procSetEvent.Call(s.event)
	}
	if s.busy {
		if reset != 0 {
			return statusErr(reset)
		}
		return nil
	}
	err := s.shutdown()
	if reset != 0 {
		return statusErr(reset)
	}
	return err
}

func (s *Stream) shutdown() error {
	prep := s.unprepare()
	var rc uint32
	if s.h != 0 {
		rc = waveOutClose(s.h)
		s.h = 0
	}
	if s.event != 0 {
		procCloseHandle.Call(s.event)
		s.event = 0
	}
	if rc != 0 {
		return statusErr(rc)
	}
	return prep
}
