//go:build linux

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
)

const (
	contextReady      = 4
	contextFailed     = 5
	contextTerminated = 6
	streamPlayback    = 1
)

type sampleSpec struct {
	format   int32
	rate     uint32
	channels uint8
}

var (
	loadOnce sync.Once
	loadErr  error

	simpleNew   func(server, name uintptr, dir int32, dev, stream uintptr, spec, chmap, attr uintptr, errno *int32) uintptr
	simpleFree  func(s uintptr)
	simpleWrite func(s uintptr, data uintptr, nbytes uintptr, errno *int32) int32
	simpleDrain func(s uintptr, errno *int32) int32
	strerror    func(code int32) uintptr

	mainloopNew      func() uintptr
	mainloopFree     func(m uintptr)
	mainloopAPI      func(m uintptr) uintptr
	mainloopPrepare  func(m uintptr, timeout int32) int32
	mainloopPoll     func(m uintptr) int32
	mainloopDispatch func(m uintptr) int32

	contextNew        func(api uintptr, name uintptr) uintptr
	contextUnref      func(c uintptr)
	contextConnect    func(c uintptr, server uintptr, flags int32, api uintptr) int32
	contextDisconnect func(c uintptr)
	contextState      func(c uintptr) int32
	contextErrno      func(c uintptr) int32
	contextSinks      func(c uintptr, cb uintptr, userdata uintptr) uintptr
	operationUnref    func(o uintptr)
)

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
	loadOnce.Do(func() { loadErr = bindAll() })
	return loadErr
}

func bindAll() error {
	simple, err := openLib("libpulse-simple.so.0")
	if err != nil {
		return err
	}
	full, err := openLib("libpulse.so.0")
	if err != nil {
		return err
	}
	binds := []struct {
		lib  uintptr
		name string
		fn   any
	}{
		{simple, "pa_simple_new", &simpleNew},
		{simple, "pa_simple_free", &simpleFree},
		{simple, "pa_simple_write", &simpleWrite},
		{simple, "pa_simple_drain", &simpleDrain},
		{full, "pa_strerror", &strerror},
		{full, "pa_mainloop_new", &mainloopNew},
		{full, "pa_mainloop_free", &mainloopFree},
		{full, "pa_mainloop_get_api", &mainloopAPI},
		{full, "pa_mainloop_prepare", &mainloopPrepare},
		{full, "pa_mainloop_poll", &mainloopPoll},
		{full, "pa_mainloop_dispatch", &mainloopDispatch},
		{full, "pa_context_new", &contextNew},
		{full, "pa_context_unref", &contextUnref},
		{full, "pa_context_connect", &contextConnect},
		{full, "pa_context_disconnect", &contextDisconnect},
		{full, "pa_context_get_state", &contextState},
		{full, "pa_context_errno", &contextErrno},
		{full, "pa_context_get_sink_info_list", &contextSinks},
		{full, "pa_operation_unref", &operationUnref},
	}
	for _, item := range binds {
		if err := bind(item.lib, item.name, item.fn); err != nil {
			return err
		}
	}
	return nil
}

func pulseErr(code int32) error {
	text := goString(strerror(code))
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
}

// Playback opens sink. An empty sink uses the server default.
// The server buffer is two periods, so Write stays on the playback clock.
// A cancelled ctx stops the next period.
func Playback(ctx context.Context, client, sink, stream string, sample Sample, rate, channels int) (*Stream, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	if client == "" {
		client = "lewkit"
	}
	if stream == "" {
		stream = "lewkit"
	}
	width := 2
	if sample == SampleF32LE {
		width = 4
	}
	frame := channels * width
	period := periodBytes(rate, frame)
	spec := sampleSpec{format: int32(sample), rate: uint32(rate), channels: uint8(channels)}
	attr := bufferAttr{
		max:    uint32(period * 2),
		target: uint32(period * 2),
		prebuf: uint32(period),
		minreq: uint32(period),
		frag:   ^uint32(0),
	}
	clientB := cString(client)
	sinkB := cString(sink)
	streamB := cString(stream)
	var errno int32
	h := simpleNew(0, ptr(clientB), streamPlayback, ptr(sinkB), ptr(streamB), uintptr(unsafe.Pointer(&spec)), 0, uintptr(unsafe.Pointer(&attr)), &errno)
	runtime.KeepAlive(clientB)
	runtime.KeepAlive(sinkB)
	runtime.KeepAlive(streamB)
	runtime.KeepAlive(&spec)
	runtime.KeepAlive(&attr)
	if h == 0 {
		return nil, pulseErr(errno)
	}
	return &Stream{ctx: ctx, h: h, frame: frame, period: period}, nil
}

type bufferAttr struct {
	max    uint32
	target uint32
	prebuf uint32
	minreq uint32
	frag   uint32
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
		rc := simpleWrite(s.h, uintptr(unsafe.Pointer(&p[0])), uintptr(n), &errno)
		runtime.KeepAlive(p)
		if rc < 0 {
			return pulseErr(errno)
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
		simpleFree(h)
		return nil
	}
	var errno int32
	rc := simpleDrain(h, &errno)
	simpleFree(h)
	if rc < 0 {
		return pulseErr(errno)
	}
	return nil
}

// List asks the server for playback sinks.
func List(ctx context.Context) ([]Sink, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	loop := mainloopNew()
	if loop == 0 {
		return nil, errMainloop
	}
	defer mainloopFree(loop)
	name := cString("lewkit")
	ctxp := contextNew(mainloopAPI(loop), ptr(name))
	runtime.KeepAlive(name)
	if ctxp == 0 {
		return nil, errContext
	}
	defer contextUnref(ctxp)
	if rc := contextConnect(ctxp, 0, 0, 0); rc < 0 {
		return nil, pulseErr(contextErrno(ctxp))
	}
	defer contextDisconnect(ctxp)
	if err := wait(ctx, loop, func() bool {
		state := contextState(ctxp)
		return state == contextReady || state == contextFailed || state == contextTerminated
	}); err != nil {
		return nil, err
	}
	if contextState(ctxp) != contextReady {
		return nil, pulseErr(contextErrno(ctxp))
	}
	sinkOnce.Do(func() {
		sinkCB = purego.NewCallback(onSink)
	})
	id := nextID.Add(1)
	state := &listState{}
	lists.Store(id, state)
	defer lists.Delete(id)
	op := contextSinks(ctxp, sinkCB, id)
	if op == 0 {
		return nil, pulseErr(contextErrno(ctxp))
	}
	defer operationUnref(op)
	if err := wait(ctx, loop, func() bool { return state.done.Load() }); err != nil {
		return nil, err
	}
	if state.fail.Load() {
		return nil, errSinkList
	}
	return append([]Sink(nil), state.sinks...), nil
}

func wait(ctx context.Context, loop uintptr, done func() bool) error {
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
		if mainloopPrepare(loop, 200) < 0 || mainloopPoll(loop) < 0 || mainloopDispatch(loop) < 0 {
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
	name := goString(*(*uintptr)(unsafe.Pointer(info)))
	description := goString(*(*uintptr)(unsafe.Pointer(info + 16)))
	state.sinks = append(state.sinks, Sink{Name: name, Description: description})
	return 0
}
