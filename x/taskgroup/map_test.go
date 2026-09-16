package taskgroup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Len(t, results, len(input))
	for i, r := range results {
		assert.Equal(t, input[i]*10, r)
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
	require.NoError(t, err)
	assert.Equal(t, 1, maxInFlight)
	require.Len(t, results, len(input))
	assert.Equal(t, input, started)
}

func TestMap_Empty(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	results, err := Map[string, string]{
		Items:    nil,
		PoolKind: IO,
		Fn:       func(context.Context, *Status, string) (string, error) { return "x", nil },
	}.Run(ctx)
	require.NoError(t, err)
	require.NotNil(t, results)
	assert.Empty(t, results)
}

func TestMap_NilFn(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	results, err := Map[int, int]{Items: []int{1}}.Run(ctx)
	assert.Empty(t, results)
	require.ErrorIs(t, err, ErrNilFn)
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
	require.NoError(t, err)
	assert.Equal(t, int64(3), saw.Load())
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
	require.Error(t, err)
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
	require.GreaterOrEqual(t, len(got), 2)
	assert.Equal(t, "plan", got[0].Name)
	assert.GreaterOrEqual(t, got[0].LiveChildren, 1)
	parent := got[0].ID
	for _, n := range got[1:] {
		assert.Equal(t, parent, n.Parent, n.Name)
	}

	close(block)
	require.NoError(t, MustFromContext(ctx).Wait())
}
