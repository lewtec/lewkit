// Package std locks the process thread and runs a job queue on it.
package std

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	uithread "github.com/lewtec/lewkit/x/driver/thread"
)

func init() { driver.Register[uithread.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "thread_std" }
func (factory) Name() string { return "Process thread" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (uithread.Driver, error) { return New(), nil }

// New returns the process-thread queue.
func New() uithread.Driver { return &backend{jobs: make(chan func(), 1)} }

type backend struct {
	bound   atomic.Bool
	running atomic.Bool
	tid     atomic.Uint64
	jobs    chan func()
	idleMu  sync.Mutex
	idles   []func()
}

func (t *backend) Bind() {
	runtime.LockOSThread()
	t.tid.Store(uithread.OSThread())
	t.bound.Store(true)
}

func (t *backend) Bound() bool { return t.bound.Load() }

func (t *backend) On() bool {
	id := uithread.OSThread()
	if id == 0 || !t.bound.Load() {
		return false
	}
	return id == t.tid.Load()
}

func (t *backend) OnIdle(fn func()) {
	t.idleMu.Lock()
	t.idles = append(t.idles, fn)
	t.idleMu.Unlock()
}

func (t *backend) Loop(ctx context.Context) {
	runtime.LockOSThread()
	t.tid.Store(uithread.OSThread())
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

func (t *backend) drain() {
	for {
		select {
		case fn := <-t.jobs:
			fn()
		default:
			return
		}
	}
}

func (t *backend) runIdle() {
	t.idleMu.Lock()
	fns := slices.Clone(t.idles)
	t.idleMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (t *backend) Do(fn func()) {
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

func (t *backend) Go(fn func()) {
	if t.On() {
		fn()
		return
	}
	t.jobs <- fn
}

func (t *backend) Enqueue(fn func()) {
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
