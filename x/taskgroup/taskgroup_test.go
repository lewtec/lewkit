package taskgroup

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NotZero(t, id)
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.True(t, ran.Load())
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
		assert.GreaterOrEqual(t, mu.Load(), int64(1), "b ran before a")
		return nil
	}, a)
	require.NoError(t, MustFromContext(ctx).Wait())
}

func TestErrorCancelsGroup(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	sentinel := errors.New("boom")
	fail := Go(ctx, "fail", CPU, func(context.Context, *Status) error {
		return sentinel
	})
	Go(ctx, "after", CPU, func(context.Context, *Status) error {
		assert.Fail(t, "should not run after dep failure")
		return nil
	}, fail)
	err := MustFromContext(ctx).Wait()
	require.ErrorIs(t, err, sentinel)
}

func TestPoolLimits(t *testing.T) {
	_, ctx := newTest(t, Limits{IO: 2, CPU: 2, Internet: 2})
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	for range 10 {
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
	}
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(2))
}

func TestUnknownDependency(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	Go(ctx, "x", CPU, func(context.Context, *Status) error {
		return nil
	}, 99)
	err := MustFromContext(ctx).Wait()
	require.ErrorIs(t, err, ErrUnknownDependency)
}

func TestStatusUnit(t *testing.T) {
	var s Status
	tnode := &task{}
	s.t = tnode
	done := s.Unit()
	assert.Equal(t, int64(0), tnode.current.Load())
	assert.Equal(t, int64(1), tnode.total.Load())
	done()
	assert.Equal(t, int64(1), tnode.current.Load())
	assert.Equal(t, int64(1), tnode.total.Load())
}

func TestFromContext(t *testing.T) {
	sess, ctx := newTest(t, DefaultLimits())
	assert.Equal(t, sess, FromContext(ctx))
	assert.Nil(t, FromContext(t.Context()))
}

func TestSessionCancelUnblocksWait(t *testing.T) {
	s, ctx := New(t.Context(), DefaultLimits())
	Go(ctx, "hold", CPU, func(ctx context.Context, _ *Status) error {
		<-ctx.Done()
		return context.Cause(ctx)
	})
	s.Cancel(nil)
	require.ErrorIs(t, s.Wait(), context.Canceled)
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
	assert.Equal(t, first, sess.Latest("setup"))
	close(block)
	require.NoError(t, sess.Wait())
	assert.Zero(t, sess.Latest("setup"))
}

func TestOnScheduleFiresAfterGo(t *testing.T) {
	s, ctx := newTest(t, DefaultLimits())
	var n atomic.Int32
	s.SetOnSchedule(func() { n.Add(1) })
	require.Equal(t, int32(0), n.Load())
	Go(ctx, "a", CPU, func(context.Context, *Status) error { return nil })
	Go(ctx, "b", CPU, func(context.Context, *Status) error { return nil })
	require.Equal(t, int32(2), n.Load())
	require.NoError(t, s.Wait())
}

func TestOnScheduleOnceFuncStartsOnce(t *testing.T) {
	s, ctx := newTest(t, DefaultLimits())
	var n atomic.Int32
	s.SetOnSchedule(sync.OnceFunc(func() { n.Add(1) }))
	Go(ctx, "a", CPU, func(context.Context, *Status) error { return nil })
	Go(ctx, "b", CPU, func(context.Context, *Status) error { return nil })
	require.Equal(t, int32(1), n.Load())
	require.NoError(t, s.Wait())
}

func TestWithSession_CreatesAndWaits(t *testing.T) {
	var ran atomic.Bool
	err := WithSession(t.Context(), func(ctx context.Context) error {
		Go(ctx, "t", CPU, func(context.Context, *Status) error {
			ran.Store(true)
			return nil
		})
		return nil
	})
	require.NoError(t, err)
	assert.True(t, ran.Load())
}

func TestWithSession_JoinsExisting(t *testing.T) {
	s, ctx := New(t.Context(), DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	err := WithSession(ctx, func(ctx context.Context) error {
		Go(ctx, "hold", CPU, func(context.Context, *Status) error {
			close(started)
			<-block
			return nil
		})
		return nil
	})
	require.NoError(t, err)
	<-started
	close(block)
	require.NoError(t, s.Wait())
}

func TestWithSession_FnErrorWins(t *testing.T) {
	boom := errors.New("boom")
	err := WithSession(t.Context(), func(ctx context.Context) error {
		Go(ctx, "t", CPU, func(context.Context, *Status) error {
			return errors.New("task")
		})
		return boom
	})
	require.ErrorIs(t, err, boom)
}

func TestWithSession_WaitError(t *testing.T) {
	boom := errors.New("boom")
	err := WithSession(t.Context(), func(ctx context.Context) error {
		Go(ctx, "t", CPU, func(context.Context, *Status) error {
			return boom
		})
		return nil
	})
	require.ErrorIs(t, err, boom)
}

func TestWithSession_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := WithSession(ctx, func(context.Context) error {
		require.FailNow(t, "fn ran")
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestWithSession_NilFn(t *testing.T) {
	require.ErrorIs(t, WithSession(t.Context(), nil), ErrNilFn)
}
