package taskgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestIsolateDoesNotCancelParentSiblings(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	release := make(chan struct{})
	var siblingRan atomic.Bool

	Go(ctx, "sibling", CPU, func(context.Context, *Status) error {
		<-release
		siblingRan.Store(true)
		return nil
	})

	isolated := errors.New("isolated failure")
	err := Isolate(ctx, func(ctx context.Context) error {
		Go(ctx, "boom", CPU, func(context.Context, *Status) error {
			return isolated
		})
		return MustFromContext(ctx).waitLive(ctx, taskFromContext(ctx))
	})
	if err == nil {
		t.Fatal("expected isolated failure")
	}
	close(release)
	if werr := MustFromContext(ctx).Wait(); werr != nil {
		t.Fatalf("parent session should succeed: %v", werr)
	}
	if !siblingRan.Load() {
		t.Fatal("parent sibling should have run")
	}
}

func TestGoIsolatedWithoutSessionRunsSync(t *testing.T) {
	var ran bool
	err := GoIsolated(t.Context(), "x", CPU, func(ctx context.Context, s *Status) error {
		ran = true
		if s == nil {
			t.Fatal("status is nil")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("fn did not run")
	}
}

func TestGoIsolatedNamedChild(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	err := GoIsolated(ctx, "install:foo", Control, func(ctx context.Context, s *Status) error {
		s.Update("done")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if werr := MustFromContext(ctx).Wait(); werr != nil {
		t.Fatal(werr)
	}
}
