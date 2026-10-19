package native

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
)

type cleanupOwner struct {
	p unsafe.Pointer
}

type cleanupHandle struct {
	ch chan struct{}
}

func (h *cleanupHandle) release() {
	if h == nil || h.ch == nil {
		return
	}
	h.ch <- struct{}{}
}

func TestArmRunsWhenUnreachable(t *testing.T) {
	ch := make(chan struct{}, 1)
	func() {
		owner := &cleanupOwner{p: unsafe.Pointer(&ch)}
		handle := &cleanupHandle{ch: ch}
		Arm(owner, (*cleanupHandle).release, handle)
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		runtime.GC()
		select {
		case <-ch:
			return
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("cleanup did not run")
		}
		runtime.Gosched()
	}
}

func TestDisarmStopsCleanup(t *testing.T) {
	ch := make(chan struct{}, 1)
	func() {
		owner := &cleanupOwner{p: unsafe.Pointer(&ch)}
		handle := &cleanupHandle{ch: ch}
		cleanup := Arm(owner, (*cleanupHandle).release, handle)
		Disarm(owner, cleanup)
	}()
	runtime.GC()
	runtime.GC()
	select {
	case <-ch:
		t.Fatal("cleanup ran after Disarm")
	default:
	}
}
