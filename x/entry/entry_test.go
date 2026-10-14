package entry

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/require"
)

var (
	errSessionOpen = errors.New("session still open")
	errBuildFailed = errors.New("build failed")
	errRan         = errors.New("ran")
	errChildFailed = errors.New("child failed")
)

func TestRunAfterViewRunsNow(t *testing.T) {
	called := false
	err := RunAfterView(t.Context(), func(context.Context) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestRunAfterViewWaitsForTheView(t *testing.T) {
	t.Cleanup(func() { After(nil) })
	var ran bool
	err := Run(t.Context(), func(ctx context.Context) error {
		return RunAfterView(ctx, func(context.Context) error {
			ran = true
			return nil
		})
	})
	require.NoError(t, err)
	require.True(t, ran)
}

func TestAfterOutlivesSession(t *testing.T) {
	t.Cleanup(func() { After(nil) })
	var sessionCtx context.Context
	err := Run(t.Context(), func(ctx context.Context) error {
		sessionCtx = ctx
		After(func(ctx context.Context) error {
			thread.Do(func() {})
			if err := ctx.Err(); err != nil {
				return err
			}
			if sessionCtx.Err() == nil {
				return errSessionOpen
			}
			return nil
		})
		return nil
	})
	require.NoError(t, err)
	require.Error(t, sessionCtx.Err())
}

func TestAfterSkipsWhenWorkFails(t *testing.T) {
	t.Cleanup(func() { After(nil) })
	err := Run(t.Context(), func(context.Context) error {
		After(func(context.Context) error {
			return errRan
		})
		return errBuildFailed
	})
	require.ErrorIs(t, err, errBuildFailed)
	require.NoError(t, Run(t.Context(), func(context.Context) error { return nil }))
}

func TestAfterError(t *testing.T) {
	t.Cleanup(func() { After(nil) })
	After(func(context.Context) error { return errChildFailed })
	err := Run(t.Context(), func(context.Context) error { return nil })
	require.ErrorIs(t, err, errChildFailed)
}

func TestAppModePanicBecomesError(t *testing.T) {
	t.Setenv("ELETROCROMO_NO_UI", "1")
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("TERM", "dumb")
	err := Run(t.Context(), func(context.Context) error {
		panic("boom")
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "panic: boom")
}

func TestRunUsesContextLimits(t *testing.T) {
	ctx := taskgroup.WithLimits(t.Context(), taskgroup.Limits{IO: 1, CPU: 2, Internet: 3})
	var got taskgroup.Limits
	err := Run(ctx, func(ctx context.Context) error {
		got = taskgroup.FromContext(ctx).Limits()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, taskgroup.Limits{IO: 1, CPU: 2, Internet: 3}, got)
}
