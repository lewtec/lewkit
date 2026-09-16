package taskgroup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMap_BasicAndOrder(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	input := []int{1, 2, 3, 4, 5}
	results, err := Map[int, int]{
		Items:    input,
		PoolKind: CPU,
		TaskName: func(i int, _ int) string { return fmt.Sprintf("map:%d", i) },
		Fn: func(ctx context.Context, s *Status, v int) (int, error) {
			s.Update(fmt.Sprintf("processing %d", v))
			return v * 10, nil
		},
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Map.Run: %v", err)
	}
	if len(results) != len(input) {
		t.Fatalf("got %d results, want %d", len(results), len(input))
	}
	for i, r := range results {
		if r != input[i]*10 {
			t.Errorf("results[%d] = %d, want %d", i, r, input[i]*10)
		}
	}
}

func TestMap_SerialRunsOneAtATime(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	var (
		mu          sync.Mutex
		inFlight    int
		maxInFlight int
		started     []int
	)
	input := []int{1, 2, 3, 4}
	results, err := Map[int, int]{
		Items:    input,
		PoolKind: CPU,
		Serial:   true,
		Fn: func(ctx context.Context, s *Status, v int) (int, error) {
			mu.Lock()
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			started = append(started, v)
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			inFlight--
			mu.Unlock()
			return v, nil
		},
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Map.Run: %v", err)
	}
	if maxInFlight != 1 {
		t.Fatalf("max concurrent %d, want 1", maxInFlight)
	}
	if len(results) != len(input) {
		t.Fatalf("results len %d, want %d", len(results), len(input))
	}
	for i, v := range started {
		if v != input[i] {
			t.Fatalf("started order %v, want %v", started, input)
		}
	}
}

func TestMap_Empty(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	results, err := Map[string, string]{
		Items:    nil,
		PoolKind: IO,
		Fn:       func(context.Context, *Status, string) (string, error) { return "x", nil },
	}.Run(ctx)
	if err != nil {
		t.Fatalf("empty Map.Run: %v", err)
	}
	if results == nil || len(results) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", results)
	}
}

func TestMap_NilFn(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	results, err := Map[int, int]{Items: []int{1}}.Run(ctx)
	if len(results) != 0 || !errors.Is(err, ErrNilFn) {
		t.Fatalf("got results=%v err=%v, want ErrNilFn", results, err)
	}
}

func TestEach_RunsWithoutResults(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	var saw atomic.Int64
	err := Each[int]{
		Name:     "each",
		Items:    []int{1, 2, 3},
		PoolKind: CPU,
		Fn: func(context.Context, *Status, int) error {
			saw.Add(1)
			return nil
		},
	}.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if saw.Load() != 3 {
		t.Fatalf("saw %d, want 3", saw.Load())
	}
}

func TestMap_ErrorPropagates(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	boom := errors.New("boom on 20")
	_, err := Map[int, int]{
		Items:    []int{10, 20, 30},
		PoolKind: IO,
		Fn: func(ctx context.Context, s *Status, v int) (int, error) {
			if v == 20 {
				return 0, boom
			}
			return v, nil
		},
	}.Run(ctx)
	if err == nil {
		t.Fatal("expected error from Map.Run")
	}
}

func TestMap_ChildrenSitUnderParent(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	var once sync.Once

	go func() {
		_, _ = Map[int, int]{
			Name:     "plan",
			Items:    []int{1, 2, 3},
			PoolKind: CPU,
			Fn: func(ctx context.Context, s *Status, v int) (int, error) {
				once.Do(func() { close(started) })
				<-block
				return v, nil
			},
		}.Run(ctx)
	}()
	<-started

	got := List(ctx, 16)
	if len(got) < 2 {
		t.Fatalf("List = %#v, want parent + children", got)
	}
	if got[0].Name != "plan" {
		t.Fatalf("parent = %q, want plan", got[0].Name)
	}
	if got[0].LiveChildren < 1 {
		t.Fatalf("parent LiveChildren = %d, want >= 1", got[0].LiveChildren)
	}
	parent := got[0].ID
	for _, n := range got[1:] {
		if n.Parent != parent {
			t.Fatalf("child %q parent = %d, want %d", n.Name, n.Parent, parent)
		}
	}

	close(block)
	if err := MustFromContext(ctx).Wait(); err != nil {
		t.Fatal(err)
	}
}
