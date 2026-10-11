//go:build linux && !android

package pulse

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/release"
)

const (
	contextReady      = 4
	contextFailed     = 5
	contextTerminated = 6
	streamPlayback    = 1
)

type pulseLibs struct {
	simple Simple
	pulse  Pulse
}

// loadPulse binds libpulse-simple and libpulse and returns both.
var loadPulse = native.Singleton(func() (pulseLibs, error) {
	simple, err := native.Bind[Simple](native.Lazy, "libpulse-simple.so.0")
	if err != nil {
		return pulseLibs{}, fmt.Errorf("libpulse-simple.so.0: %w", err)
	}
	pulse, err := native.Bind[Pulse](native.Lazy, "libpulse.so.0")
	if err != nil {
		return pulseLibs{}, fmt.Errorf("libpulse.so.0: %w", err)
	}
	return pulseLibs{simple: simple, pulse: pulse}, nil
})

type listState struct {
	sinks []Sink
	done  atomic.Bool
	fail  atomic.Bool
}

var (
	lists    sync.Map
	nextID   atomic.Uintptr
	sinkCB   uintptr
	sinkOnce sync.Once
)

// Available loads libpulse-simple and libpulse.
func Available() error {
	_, err := loadPulse()
	return err
}

func pulseErr(lib Pulse, code int32) error {
	text := native.GoString(lib.PaStrerror(code))
	if text == "" {
		return errUnavailable
	}
	return fmt.Errorf("%w: %s", errUnavailable, text)
}

// Stream plays PCM through pa_simple.
type Stream struct {
	ctx    context.Context
	h      uintptr
	frame  int
	period int
	simple Simple
	pulse  Pulse
}

// Playback opens sink. An empty sink uses the server default.
// The server buffer is two periods, so Write stays on the playback clock.
// A cancelled ctx stops the next period.
func Playback(ctx context.Context, client, sink, stream string, sample Sample, rate, channels int) (*Stream, error) {
	libs, err := loadPulse()
	if err != nil {
		return nil, err
	}
	if client == "" {
		client = release.Name()
	}
	if stream == "" {
		stream = release.Name()
	}
	width := 2
	if sample == SampleF32LE {
		width = 4
	}
	frame := channels * width
	period := periodBytes(rate, frame)
	spec := SampleSpec{Format: int32(sample), Rate: uint32(rate), Channels: uint8(channels)}
	attr := BufferAttr{
		Max:    uint32(period * 2),
		Target: uint32(period * 2),
		Prebuf: uint32(period),
		Minreq: uint32(period),
		Frag:   ^uint32(0),
	}
	clientB := native.CString(client)
	var sinkB []byte
	if sink != "" {
		sinkB = native.CString(sink)
	}
	streamB := native.CString(stream)
	var errno int32
	h := libs.simple.PaSimpleNew(0, ptr(clientB), streamPlayback, ptr(sinkB), ptr(streamB), &spec, 0, &attr, &errno)
	runtime.KeepAlive(clientB)
	runtime.KeepAlive(sinkB)
	runtime.KeepAlive(streamB)
	runtime.KeepAlive(&spec)
	runtime.KeepAlive(&attr)
	if h == 0 {
		return nil, pulseErr(libs.pulse, errno)
	}
	return &Stream{ctx: ctx, h: h, frame: frame, period: period, simple: libs.simple, pulse: libs.pulse}, nil
}

func periodBytes(rate, frame int) int {
	n := rate * frame / 50
	if n < frame {
		n = frame
	}
	return n - n%frame
}

// Write plays p one period at a time. The Pulse buffer holds two periods.
func (s *Stream) Write(p []byte) error {
	if s == nil || s.h == 0 {
		return errClosed
	}
	for len(p) > 0 {
		if err := s.ctx.Err(); err != nil {
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
		var errno int32
		rc := s.simple.PaSimpleWrite(s.h, uintptr(unsafe.Pointer(&p[0])), uintptr(n), &errno)
		runtime.KeepAlive(p)
		if rc < 0 {
			return pulseErr(s.pulse, errno)
		}
		p = p[n:]
	}
	return nil
}

// Close drains queued PCM and frees the stream.
func (s *Stream) Close() error {
	if s == nil || s.h == 0 {
		return nil
	}
	h := s.h
	s.h = 0
	if s.ctx.Err() != nil {
		s.simple.PaSimpleFree(h)
		return nil
	}
	var errno int32
	rc := s.simple.PaSimpleDrain(h, &errno)
	s.simple.PaSimpleFree(h)
	if rc < 0 {
		return pulseErr(s.pulse, errno)
	}
	return nil
}

// List asks the server for playback sinks.
func List(ctx context.Context) ([]Sink, error) {
	libs, err := loadPulse()
	if err != nil {
		return nil, err
	}
	pulse := libs.pulse
	loop := pulse.PaMainloopNew()
	if loop == 0 {
		return nil, errMainloop
	}
	defer pulse.PaMainloopFree(loop)
	name := native.CString(release.Name())
	ctxp := pulse.PaContextNew(pulse.PaMainloopGetApi(loop), ptr(name))
	runtime.KeepAlive(name)
	if ctxp == 0 {
		return nil, errContext
	}
	defer pulse.PaContextUnref(ctxp)
	if rc := pulse.PaContextConnect(ctxp, 0, 0, 0); rc < 0 {
		return nil, pulseErr(pulse, pulse.PaContextErrno(ctxp))
	}
	defer pulse.PaContextDisconnect(ctxp)
	if err := wait(ctx, pulse, loop, func() bool {
		state := pulse.PaContextGetState(ctxp)
		return state == contextReady || state == contextFailed || state == contextTerminated
	}); err != nil {
		return nil, err
	}
	if pulse.PaContextGetState(ctxp) != contextReady {
		return nil, pulseErr(pulse, pulse.PaContextErrno(ctxp))
	}
	sinkOnce.Do(func() {
		sinkCB = purego.NewCallback(onSink)
	})
	id := nextID.Add(1)
	state := &listState{}
	lists.Store(id, state)
	defer lists.Delete(id)
	op := pulse.PaContextGetSinkInfoList(ctxp, sinkCB, id)
	if op == 0 {
		return nil, pulseErr(pulse, pulse.PaContextErrno(ctxp))
	}
	defer pulse.PaOperationUnref(op)
	if err := wait(ctx, pulse, loop, func() bool { return state.done.Load() }); err != nil {
		return nil, err
	}
	if state.fail.Load() {
		return nil, errSinkList
	}
	return append([]Sink(nil), state.sinks...), nil
}

func wait(ctx context.Context, pulse Pulse, loop uintptr, done func() bool) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if done() {
			return nil
		}
		if time.Now().After(deadline) {
			return errTimeout
		}
		if pulse.PaMainloopPrepare(loop, 200) < 0 || pulse.PaMainloopPoll(loop) < 0 || pulse.PaMainloopDispatch(loop) < 0 {
			return errMainloop
		}
	}
}

func onSink(_, info, eol, userdata uintptr) uintptr {
	value, ok := lists.Load(userdata)
	if !ok {
		return 0
	}
	state := value.(*listState)
	if int32(eol) != 0 {
		if int32(eol) < 0 {
			state.fail.Store(true)
		}
		state.done.Store(true)
		return 0
	}
	if info == 0 {
		return 0
	}
	name := native.GoString(*(*uintptr)(unsafe.Pointer(info)))
	description := native.GoString(*(*uintptr)(unsafe.Pointer(info + 16)))
	state.sinks = append(state.sinks, Sink{Name: name, Description: description})
	return 0
}
