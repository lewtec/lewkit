package taskgroup

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestListEmpty(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	if got := List(ctx, 8); got != nil {
		t.Fatalf("List empty session = %#v, want nil", got)
	}
	if got := List(ctx, 0); got != nil {
		t.Fatalf("List(0) = %#v, want nil", got)
	}
}

func TestListWalksTreeUntilFull(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	const n = 10_000

	Go(ctx, "apply", Control, func(ctx context.Context, s *Status) error {
		s.Progress(0, n)
		for i := range n {
			Go(ctx, "file", IO, func(context.Context, *Status) error {
				<-block
				return nil
			})
			_ = i
		}
		close(started)
		<-block
		return nil
	})
	<-started

	got := List(ctx, 8)
	if len(got) != 8 {
		t.Fatalf("List(8) len = %d, want 8", len(got))
	}
	if got[0].Name != "apply" {
		t.Fatalf("first row = %q, want apply", got[0].Name)
	}
	if got[0].LiveChildren != n {
		t.Fatalf("apply LiveChildren = %d, want %d", got[0].LiveChildren, n)
	}
	if got[0].Parent != 0 {
		t.Fatalf("top-level Parent = %d, want 0", got[0].Parent)
	}
	parent := got[0].ID
	for i, n := range got[1:] {
		if n.Name != "file" {
			t.Fatalf("row %d name = %q, want file", i+1, n.Name)
		}
		if n.Parent != parent {
			t.Fatalf("row %d parent = %d, want %d", i+1, n.Parent, parent)
		}
	}

	close(block)
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatal(err)
	}
	if got := List(ctx, 8); len(got) != 0 {
		t.Fatalf("List after Wait len = %d, want 0", len(got))
	}
}

func TestListKeepsFinishedParentWhileChildrenRun(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})

	Go(ctx, "apply", Control, func(ctx context.Context, _ *Status) error {
		Go(ctx, "file", IO, func(context.Context, *Status) error {
			close(started)
			<-block
			return nil
		})
		return nil
	})
	<-started

	var got []Node
	deadline := time.Now().Add(time.Second)
	for {
		got = List(ctx, 8)
		if len(got) >= 2 && got[0].Name == "apply" && got[0].State == Done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("List = %#v, want done apply above file", got)
		}
		time.Sleep(time.Millisecond)
	}
	if got[0].LiveChildren != 1 {
		t.Fatalf("apply LiveChildren = %d, want 1", got[0].LiveChildren)
	}

	close(block)
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatal(err)
	}
	if got := List(ctx, 8); len(got) != 0 {
		t.Fatalf("List after Wait len = %d, want 0", len(got))
	}
}

func TestListSkipsIsolateBoundary(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	var saw atomic.Bool

	Go(ctx, "bundle", Control, func(ctx context.Context, _ *Status) error {
		return Isolate(ctx, func(ctx context.Context) error {
			Go(ctx, "bundle:icons", CPU, func(context.Context, *Status) error {
				saw.Store(true)
				close(started)
				<-block
				return nil
			})
			return MustFromContext(ctx).waitLive(ctx, taskFromContext(ctx))
		})
	})
	<-started

	got := List(ctx, 16)
	names := make([]string, len(got))
	for i, n := range got {
		names[i] = n.Name
	}
	for _, n := range got {
		if n.Name == "" {
			t.Fatalf("List included empty-name node: %#v", got)
		}
	}
	if len(got) < 2 || names[0] != "bundle" || names[1] != "bundle:icons" {
		t.Fatalf("List names = %v, want [bundle bundle:icons …]", names)
	}
	if got[1].Parent != got[0].ID {
		t.Fatalf("icons parent = %d, want bundle %d", got[1].Parent, got[0].ID)
	}

	close(block)
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatal(err)
	}
	if !saw.Load() {
		t.Fatal("isolated child did not run")
	}
}

func TestListNilContext(t *testing.T) {
	if got := List(t.Context(), 4); got != nil {
		t.Fatalf("List without session = %#v", got)
	}
}
