//go:build darwin

package coreaudio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	cfUTF8  = 0x08000100
	periods = 2
)

var (
	errClosed    = errors.New("coreaudio stream closed")
	errCoreAudio = errors.New("coreaudio")
)

// audioLibs is AudioToolbox, CoreAudio, and CoreFoundation from one load.
type audioLibs struct {
	audio AudioToolbox
	core  CoreAudio
	cf    CoreFoundation
}

// loadAudio binds the three frameworks and returns them from that same call.
var loadAudio = native.Singleton(func() (audioLibs, error) {
	audio, err := native.Bind[AudioToolbox](native.Lazy|native.Global, "/System/Library/Frameworks/AudioToolbox.framework/AudioToolbox")
	if err != nil {
		return audioLibs{}, err
	}
	core, err := native.Bind[CoreAudio](native.Lazy|native.Global, "/System/Library/Frameworks/CoreAudio.framework/CoreAudio")
	if err != nil {
		return audioLibs{}, err
	}
	cf, err := native.Bind[CoreFoundation](native.Lazy|native.Global, "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
	if err != nil {
		return audioLibs{}, err
	}
	return audioLibs{audio: audio, core: core, cf: cf}, nil
})

// Available loads AudioToolbox, CoreAudio, and CoreFoundation.
func Available() error {
	_, err := loadAudio()
	return err
}

func statusErr(st int32) error {
	if st == 0 {
		return nil
	}
	return fmt.Errorf("%w: %d", errCoreAudio, st)
}

func cfString(cf CoreFoundation, ref uintptr) string {
	if ref == 0 || cf.CFStringGetLength == nil || cf.CFStringGetCString == nil {
		return ""
	}
	n := cf.CFStringGetLength(ref)
	if n < 0 {
		n = 0
	}
	buf := make([]byte, n*4+1)
	if cf.CFStringGetCString(ref, uintptr(unsafe.Pointer(&buf[0])), int64(len(buf)), cfUTF8) == 0 {
		return ""
	}
	end := bytes.IndexByte(buf, 0)
	if end < 0 {
		end = len(buf)
	}
	return string(buf[:end])
}

func prop(core CoreAudio, id uint32, addr propAddr, out []byte) error {
	size := uint32(len(out))
	st := core.AudioObjectGetPropertyData(id, uintptr(unsafe.Pointer(&addr)), 0, 0, &size, uintptr(unsafe.Pointer(&out[0])))
	return statusErr(st)
}

func propString(libs audioLibs, id uint32, selector uint32) (string, error) {
	addr := propAddr{selector: selector, scope: fourcc("glob"), element: 0}
	var ref uintptr
	size := uint32(unsafe.Sizeof(ref))
	st := libs.core.AudioObjectGetPropertyData(id, uintptr(unsafe.Pointer(&addr)), 0, 0, &size, uintptr(unsafe.Pointer(&ref)))
	if st != 0 {
		return "", statusErr(st)
	}
	defer libs.cf.CFRelease(ref)
	return cfString(libs.cf, ref), nil
}

func outputChannels(libs audioLibs, id uint32) (int, error) {
	addr := propAddr{selector: fourcc("slay"), scope: fourcc("outp"), element: 0}
	var size uint32
	if st := libs.core.AudioObjectGetPropertyDataSize(id, uintptr(unsafe.Pointer(&addr)), 0, 0, &size); st != 0 {
		return 0, statusErr(st)
	}
	if size < 8 {
		return 0, nil
	}
	buf := make([]byte, size)
	if err := prop(libs.core, id, addr, buf); err != nil {
		return 0, err
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	total := 0
	for i := uint32(0); i < n; i++ {
		off := 8 + int(i)*16
		if off+4 > len(buf) {
			break
		}
		total += int(*(*uint32)(unsafe.Pointer(&buf[off])))
	}
	return total, nil
}

// Devices lists output devices.
func Devices() ([]Device, error) {
	libs, err := loadAudio()
	if err != nil {
		return nil, err
	}
	addr := propAddr{selector: fourcc("dev#"), scope: fourcc("glob"), element: 0}
	var size uint32
	if st := libs.core.AudioObjectGetPropertyDataSize(1, uintptr(unsafe.Pointer(&addr)), 0, 0, &size); st != 0 {
		return nil, statusErr(st)
	}
	if size < 4 {
		return nil, nil
	}
	ids := make([]uint32, size/4)
	raw := unsafe.Slice((*byte)(unsafe.Pointer(&ids[0])), len(ids)*4)
	if err := prop(libs.core, 1, addr, raw); err != nil {
		return nil, err
	}
	var out []Device
	for _, id := range ids {
		channels, err := outputChannels(libs, id)
		if err != nil || channels == 0 {
			continue
		}
		uid, err := propString(libs, id, fourcc("uid "))
		if err != nil || uid == "" {
			continue
		}
		name, err := propString(libs, id, fourcc("lnam"))
		if err != nil || name == "" {
			name = uid
		}
		out = append(out, Device{ID: uid, Name: name})
	}
	return out, nil
}

// Stream plays PCM through one AudioQueue.
type Stream struct {
	mu      sync.Mutex
	ctx     context.Context
	q       uintptr
	frame   int
	period  int
	free    chan uintptr
	id      uintptr
	started bool
	closed  bool
	libs    audioLibs
}

var (
	streams sync.Map
	nextID  atomic.Uintptr
	cbOnce  sync.Once
	queueCB uintptr
)

func onQueue(user, _, buf uintptr) uintptr {
	if value, ok := streams.Load(user); ok {
		ch := value.(chan uintptr)
		select {
		case ch <- buf:
		default:
		}
	}
	return 0
}

// Open plays layout on uid. An empty uid uses the default output device.
// Write stays one period behind the device clock. A cancelled ctx stops playback.
func Open(ctx context.Context, uid string, layout Layout) (*Stream, error) {
	libs, err := loadAudio()
	if err != nil {
		return nil, err
	}
	width := 2
	flags := uint32(12) // signed | packed
	if layout.Sample == SampleF32LE {
		width = 4
		flags = 9 // float | packed
	}
	frame := layout.Channels * width
	desc := streamDesc{
		rate:            float64(layout.Rate),
		formatID:        fourcc("lpcm"),
		flags:           flags,
		bytesPerPacket:  uint32(frame),
		framesPerPacket: 1,
		bytesPerFrame:   uint32(frame),
		channels:        uint32(layout.Channels),
		bits:            uint32(width * 8),
	}
	cbOnce.Do(func() { queueCB = purego.NewCallback(onQueue) })
	id := nextID.Add(1)
	var q uintptr
	st := libs.audio.AudioQueueNewOutput(uintptr(unsafe.Pointer(&desc)), queueCB, id, 0, 0, 0, &q)
	if st != 0 {
		return nil, statusErr(st)
	}
	if uid != "" {
		raw := native.CString(uid)
		ref := libs.cf.CFStringCreateWithCString(0, uintptr(unsafe.Pointer(&raw[0])), cfUTF8)
		if ref == 0 {
			libs.audio.AudioQueueDispose(q, 1)
			return nil, errCoreAudio
		}
		st = libs.audio.AudioQueueSetProperty(q, fourcc("aqcd"), uintptr(unsafe.Pointer(&ref)), uint32(unsafe.Sizeof(ref)))
		libs.cf.CFRelease(ref)
		if st != 0 {
			libs.audio.AudioQueueDispose(q, 1)
			return nil, statusErr(st)
		}
	}
	period := periodBytes(layout.Rate, frame)
	free := make(chan uintptr, periods)
	for range periods {
		var buf uintptr
		if st = libs.audio.AudioQueueAllocateBuffer(q, uint32(period), &buf); st != 0 {
			libs.audio.AudioQueueDispose(q, 1)
			return nil, statusErr(st)
		}
		free <- buf
	}
	s := &Stream{ctx: ctx, q: q, frame: frame, period: period, free: free, id: id, libs: libs}
	streams.Store(id, free)
	return s, nil
}

// Write feeds the device one period at a time and blocks while both periods
// are still queued. The caller stays on the device clock.
func (s *Stream) Write(p []byte) error {
	if s == nil {
		return errClosed
	}
	for len(p) > 0 {
		buf, err := s.acquire()
		if err != nil {
			return err
		}
		n := len(p)
		if n > s.period {
			n = s.period
		}
		n -= n % s.frame
		if n == 0 {
			return nil
		}
		fillQueueBuffer(buf, p[:n])
		s.mu.Lock()
		if s.closed || s.q == 0 {
			s.mu.Unlock()
			return errClosed
		}
		if st := s.libs.audio.AudioQueueEnqueueBuffer(s.q, buf, 0, 0); st != 0 {
			s.mu.Unlock()
			return statusErr(st)
		}
		if !s.started {
			if st := s.libs.audio.AudioQueueStart(s.q, 0); st != 0 {
				s.mu.Unlock()
				return statusErr(st)
			}
			s.started = true
		}
		s.mu.Unlock()
		p = p[n:]
	}
	return nil
}

func (s *Stream) acquire() (uintptr, error) {
	select {
	case <-s.ctx.Done():
		if err := s.halt(true); err != nil {
			return 0, err
		}
		return 0, s.ctx.Err()
	case buf := <-s.free:
		if buf == 0 {
			if err := s.ctx.Err(); err != nil {
				return 0, err
			}
			return 0, errClosed
		}
		return buf, nil
	}
}

// Close plays the queued period out and releases the queue.
func (s *Stream) Close() error {
	if s == nil {
		return nil
	}
	immediate := s.ctx.Err() != nil
	return s.halt(immediate)
}

func (s *Stream) halt(immediate bool) error {
	s.mu.Lock()
	if s.q == 0 {
		s.closed = true
		s.mu.Unlock()
		s.wake()
		return nil
	}
	s.closed = true
	q := s.q
	started := s.started
	s.q = 0
	s.mu.Unlock()
	s.wake()
	streams.Delete(s.id)
	flag := byte(0)
	if immediate {
		flag = 1
	}
	var err error
	if started {
		if st := s.libs.audio.AudioQueueStop(q, flag); st != 0 {
			err = statusErr(st)
		}
	}
	if st := s.libs.audio.AudioQueueDispose(q, 1); st != 0 && err == nil {
		err = statusErr(st)
	}
	return err
}

func (s *Stream) wake() {
	select {
	case s.free <- 0:
	default:
	}
}

func fillQueueBuffer(buf uintptr, p []byte) {
	data := *(*uintptr)(unsafe.Pointer(buf + 8))
	copy(unsafe.Slice((*byte)(unsafe.Pointer(data)), len(p)), p)
	*(*uint32)(unsafe.Pointer(buf + 16)) = uint32(len(p))
}

func periodBytes(rate, frame int) int {
	n := rate * frame / 50
	if n < frame {
		n = frame
	}
	return n - n%frame
}
