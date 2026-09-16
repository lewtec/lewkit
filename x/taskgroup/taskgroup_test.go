package taskgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func newTest(t *testing.T, limits Limits) (*Session, context.Context) {
	t.Helper()
	s, ctx := New(t.Context(), limits)
	t.Cleanup(func() {
		if err := s.Wait(); err != nil {
			t.Logf("Wait: %v", err)
		}
	})
	return s, ctx
}

func TestBasicExecution(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	var ran atomic.Bool
	id := Go(ctx, "task1", CPU, func(context.Context, *Status) error {
		ran.Store(true)
		return nil
	})
	if id == 0 {
		t.Fatal("Go returned 0")
	}
	s := MustFromContext(ctx)
	if err := s.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !ran.Load() {
		t.Fatal("task did not run")
	}
}

func TestDependencyOrder(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	var mu atomic.Int64
	a := Go(ctx, "a", CPU, func(context.Context, *Status) error {
		time.Sleep(10 * time.Millisecond)
		mu.Add(1)
		return nil
	})
	Go(ctx, "b", CPU, func(context.Context, *Status) error {
		if mu.Load() < 1 {
			t.Error("b ran before a")
		}
		return nil
	}, a)
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
}

func TestErrorCancelsGroup(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	sentinel := errors.New("boom")
	fail := Go(ctx, "fail", CPU, func(context.Context, *Status) error {
		return sentinel
	})
	Go(ctx, "after", CPU, func(context.Context, *Status) error {
		t.Error("should not run after dep failure")
		return nil
	}, fail)
	err := MustFromContext(ctx).Wait()
	if !errors.Is(err, sentinel) {
		t.Fatalf("Wait = %v, want %v", err, sentinel)
	}
}

func TestPoolLimits(t *testing.T) {
	_, ctx := newTest(t, Limits{IO: 2, CPU: 2, Internet: 2})
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	for i := range 10 {
		Go(ctx, "t", IO, func(context.Context, *Status) error {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		})
		_ = i
	}
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if maxConcurrent.Load() > 2 {
		t.Fatalf("max concurrent %d exceeded pool limit 2", maxConcurrent.Load())
	}
}

func TestUnknownDependency(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	Go(ctx, "x", CPU, func(context.Context, *Status) error {
		return nil
	}, 99)
	err := MustFromContext(ctx).Wait()
	if !errors.Is(err, ErrUnknownDependency) {
		t.Fatalf("Wait = %v, want ErrUnknownDependency", err)
	}
}

func TestStatusUnit(t *testing.T) {
	var s Status
	tnode := &task{}
	s.t = tnode
	done := s.Unit()
	if tnode.current.Load() != 0 || tnode.total.Load() != 1 {
		t.Fatalf("Unit start: cur=%d total=%d", tnode.current.Load(), tnode.total.Load())
	}
	done()
	if tnode.current.Load() != 1 || tnode.total.Load() != 1 {
		t.Fatalf("Unit done: cur=%d total=%d", tnode.current.Load(), tnode.total.Load())
	}
}

func TestFromContext(t *testing.T) {
	sess, ctx := newTest(t, DefaultLimits())
	if FromContext(ctx) != sess {
		t.Fatal("FromContext did not return the session")
	}
	if FromContext(t.Context()) != nil {
		t.Fatal("FromContext on empty context should return nil")
	}
}

func TestLatestByName(t *testing.T) {
	sess, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	first := Go(ctx, "setup", CPU, func(context.Context, *Status) error {
		close(started)
		<-block
		return nil
	})
	<-started
	if got := sess.Latest("setup"); got != first {
		t.Fatalf("Latest = %d, want %d", got, first)
	}
	close(block)
	if err := sess.Wait(); err != nil {
		t.Fatal(err)
	}
	if got := sess.Latest("setup"); got != 0 {
		t.Fatalf("Latest after finish = %d, want 0", got)
	}
}
