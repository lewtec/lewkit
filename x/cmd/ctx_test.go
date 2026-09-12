package cmd

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAppFlags(t *testing.T) {
	app, err := Parse[App[None]]("-vv", "--profile-dir", "/tmp/p")
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&app).Elem())
	assert.Equal(t, 2, Get[int](ctx, "verbose"))
	assert.Equal(t, "/tmp/p", Get[string](ctx, "profile-dir"))
	assert.False(t, Get[bool](ctx, "help"))
	assert.False(t, Get[bool](ctx, "version"))
}

type ctxLeaf struct {
	got int
	dir string
}

func (l *ctxLeaf) Run(ctx context.Context) error {
	l.got = Get[int](ctx, "verbose")
	l.dir = Get[string](ctx, "profile-dir")
	return nil
}

type ctxRoot struct {
	leaf *ctxLeaf
}

func TestGetParentFlagAfterCommand(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	app, err := Parse[App[ctxRoot]]("leaf", "-vv", "--profile-dir", "/tmp/x")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
	require.NotNil(t, app.Args.leaf)
	assert.Equal(t, 2, app.Args.leaf.got)
	assert.Equal(t, "/tmp/x", app.Args.leaf.dir)
}

func TestGetUnwrapsArgAndSlice(t *testing.T) {
	type args struct {
		name StringArg   `long:"name" default:"" ctx:"name"`
		tag  []StringArg `long:"tag" ctx:"tag"`
		n    Count       `long:"n" ctx:"n"`
		ok   Flag        `long:"ok" ctx:"ok"`
	}
	got, err := Parse[args]("--name", "Ada", "--tag", "a", "--tag", "b", "--n", "3", "--ok")
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, "Ada", Get[string](ctx, "name"))
	assert.Equal(t, []string{"a", "b"}, Get[[]string](ctx, "tag"))
	assert.Equal(t, 3, Get[int](ctx, "n"))
	assert.True(t, Get[bool](ctx, "ok"))
}

func TestEmptyCtxUsesLong(t *testing.T) {
	type args struct {
		name StringArg `long:"name" default:"" ctx:""`
	}
	got, err := Parse[args]("--name", "Ada")
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, "Ada", Get[string](ctx, "name"))
}

func TestEmptyCtxUsesFieldName(t *testing.T) {
	type args struct {
		path StringArg `ctx:"" default:"."`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, ".", Get[string](ctx, "path"))
}

func TestUntaggedNotInBag(t *testing.T) {
	type args struct {
		name StringArg `long:"name" default:""`
	}
	got, err := Parse[args]("--name", "Ada")
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assertPanicIs(t, ErrNotSet, func() {
		Get[string](ctx, "name")
	})
}

func TestDuplicateCtx(t *testing.T) {
	type args struct {
		a StringArg `long:"a" default:"" ctx:"x"`
		b StringArg `long:"b" default:"" ctx:"x"`
	}
	_, err := Parse[args]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestGetMissing(t *testing.T) {
	ctx := withValues(t.Context())
	assertPanicIs(t, ErrNotSet, func() {
		Get[int](ctx, "nope")
	})
}

func TestGetWrongType(t *testing.T) {
	ctx := withValues(t.Context())
	put(ctx, "verbose", 2)
	assertPanicIs(t, ErrWrongType, func() {
		Get[string](ctx, "verbose")
	})
}

func TestGetNoBag(t *testing.T) {
	assertPanicIs(t, ErrNoValues, func() {
		Get[int](t.Context(), "verbose")
	})
}

func TestNilPointerSkipped(t *testing.T) {
	type args struct {
		name *StringArg `ctx:"name"`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assertPanicIs(t, ErrNotSet, func() {
		Get[string](ctx, "name")
	})
}

func assertPanicIs(t *testing.T, want error, fn func()) {
	t.Helper()
	var got any
	func() {
		defer func() { got = recover() }()
		fn()
	}()
	require.NotNil(t, got)
	err, ok := got.(error)
	require.True(t, ok)
	assert.ErrorIs(t, err, want)
}
