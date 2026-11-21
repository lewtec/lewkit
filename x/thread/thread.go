// Package thread pins one goroutine to an OS thread and runs work there.
//
// init locks the main goroutine to the process main OS thread (AppKit
// nextEvent requires that). Run from main; Loop stays on that thread.
// Other goroutines call Do or Go.
//
// The scheduler will not move a goroutine onto a chosen OS thread. Yielding
// until you "land" on main does not work. LockOSThread only holds the
// current thread; send work to the goroutine that bound it.
package thread

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Thread is one locked OS thread plus a job queue.
type Thread struct {
	bound   atomic.Bool
	running atomic.Bool
	tid     atomic.Uint64
	jobs    chan func()
	idleMu  sync.Mutex
	idles   []func()
}

// New returns a thread that is not bound yet.
func New() *Thread {
	// Cap 1: Go can return while Loop is inside an AppKit call instead of
	// blocking the painter until the next NS event.
	return &Thread{jobs: make(chan func(), 1)}
}

var mainThread = New()

// Bind locks this goroutine to its OS thread. Call from init or main.
func Bind() { mainThread.Bind() }

// Bound reports whether Bind has been called on the process thread.
func Bound() bool { return mainThread.Bound() }

// On reports whether this goroutine is on the process bound thread.
func On() bool { return mainThread.On() }

// OnIdle registers fn on the process thread when Loop has no job.
func OnIdle(fn func()) { mainThread.OnIdle(fn) }

// Loop serves the process thread until ctx is done.
func Loop(ctx context.Context) { mainThread.Loop(ctx) }

// Do runs fn on the process bound thread and waits.
func Do(fn func()) { mainThread.Do(fn) }

// Go queues fn on the process bound thread and returns.
func Go(fn func()) { mainThread.Go(fn) }

// Enqueue queues fn on the process bound thread without blocking the caller.
func Enqueue(fn func()) { mainThread.Enqueue(fn) }

// Bind locks this goroutine to its OS thread.
func (t *Thread) Bind() {
	runtime.LockOSThread()
	t.tid.Store(osThread())
	t.bound.Store(true)
}

// Bound reports whether Bind or Loop has pinned this Thread.
func (t *Thread) Bound() bool { return t.bound.Load() }

// On reports whether this goroutine is on t's OS thread.
func (t *Thread) On() bool {
	id := osThread()
	if id == 0 || !t.bound.Load() {
		return false
	}
	return id == t.tid.Load()
}

// OnIdle registers fn to run when Loop has no job. fn may block briefly.
func (t *Thread) OnIdle(fn func()) {
	t.idleMu.Lock()
	t.idles = append(t.idles, fn)
	t.idleMu.Unlock()
}

// Loop serves Do/Go on this OS thread until ctx is done.
func (t *Thread) Loop(ctx context.Context) {
	runtime.LockOSThread()
	t.tid.Store(osThread())
	t.bound.Store(true)
	t.running.Store(true)
	defer t.running.Store(false)
	wait := time.NewTicker(2 * time.Millisecond)
	defer wait.Stop()
	for {
		t.drain()
		if ctx.Err() != nil {
			return
		}
		t.runIdle()
		select {
		case <-ctx.Done():
			return
		case fn := <-t.jobs:
			fn()
		case <-wait.C:
		}
	}
}

func (t *Thread) drain() {
	for {
		select {
		case fn := <-t.jobs:
			fn()
		default:
			return
		}
	}
}

func (t *Thread) runIdle() bool {
	t.idleMu.Lock()
	fns := slices.Clone(t.idles)
	t.idleMu.Unlock()
	if len(fns) == 0 {
		return false
	}
	for _, fn := range fns {
		fn()
	}
	return true
}

// Do runs fn on t and waits. Nested Do on the same thread runs inline.
func (t *Thread) Do(fn func()) {
	if t.On() {
		fn()
		return
	}
	done := make(chan struct{})
	t.jobs <- func() {
		fn()
		close(done)
	}
	<-done
}

// Go queues fn on t and returns. On t it runs inline.
func (t *Thread) Go(fn func()) {
	if t.On() {
		fn()
		return
	}
	t.jobs <- fn
}

// Enqueue queues fn without blocking the caller. If the job buffer is
// full, a helper goroutine waits so the painter can keep running.
func (t *Thread) Enqueue(fn func()) {
	if t.On() {
		fn()
		return
	}
	select {
	case t.jobs <- fn:
	default:
		go func() { t.jobs <- fn }()
	}
}
