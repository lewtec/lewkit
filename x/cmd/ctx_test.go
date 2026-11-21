package cmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flattenPool struct {
	io  IntArg[int] `long:"io" default:"0"`
	cpu IntArg[int] `long:"cpu" default:"0"`
}

func (p flattenPool) Value() int {
	return p.io.Value() + p.cpu.Value()
}

func TestFlattenCtxStoresValue(t *testing.T) {
	type args struct {
		flattenPool `flatten:"" ctx:"pool"`
	}
	got := ParseOK[args](t, "--io", "3", "--cpu", "4")
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, 7, Get[int](ctx, "pool"))
	assert.Panics(t, func() { Get[int](ctx, "io") })
}

type nilPtrValuer struct {
	n IntArg[int] `long:"n" default:"0"`
	p *string
}

func (v nilPtrValuer) Value() *string { return v.p }

func TestFlattenCtxStoresFieldWhenValueNil(t *testing.T) {
	type args struct {
		nilPtrValuer `flatten:"" ctx:"obj"`
	}
	got := ParseOK[args](t, "--n", "1")
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	obj := Get[nilPtrValuer](ctx, "obj")
	assert.Equal(t, 1, obj.n.Value())
	assert.Nil(t, obj.Value())
}

func TestFlattenCtxDuplicate(t *testing.T) {
	type args struct {
		a flattenPool `flatten:"" ctx:"pool"`
		b StringArg   `long:"name" default:"" ctx:"pool"`
	}
	_, err := Parse[args]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestLookupMissing(t *testing.T) {
	_, ok := Lookup[int](t.Context(), "verbose")
	assert.False(t, ok)
}

func TestGetAppFlags(t *testing.T) {
	app := ParseOK[App[None]](t, "-vv", "--pprof", "/tmp/p")
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&app).Elem())
	assert.Equal(t, 2, Get[int](ctx, "verbose"))
	assert.Equal(t, "/tmp/p", Get[string](ctx, "pprof"))
	assert.False(t, Get[bool](ctx, "help"))
	assert.False(t, Get[bool](ctx, "version"))
}

type ctxLeaf struct {
	got int
	dir string
}

func (l *ctxLeaf) Run(ctx context.Context) error {
	l.got = Get[int](ctx, "verbose")
	l.dir = Get[string](ctx, "pprof")
	return nil
}

type ctxRoot struct {
	leaf *ctxLeaf
}

func TestGetParentFlagAfterCommand(t *testing.T) {
	test.RestoreSlog(t)

	app := ParseOK[App[ctxRoot]](t, "leaf", "-vv", "--pprof", "/tmp/x")
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
	got := ParseOK[args](t, "--name", "Ada", "--tag", "a", "--tag", "b", "--n", "3", "--ok")
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
	got := ParseOK[args](t, "--name", "Ada")
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, "Ada", Get[string](ctx, "name"))
}

func TestEmptyCtxUsesFieldName(t *testing.T) {
	type args struct {
		path StringArg `ctx:"" default:"."`
	}
	got := ParseOK[args](t)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.Equal(t, ".", Get[string](ctx, "path"))
}

func TestUntaggedNotInBag(t *testing.T) {
	type args struct {
		name StringArg `long:"name" default:""`
	}
	got := ParseOK[args](t, "--name", "Ada")
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.PanicsWithError(t, "context key not set: name", func() {
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
	assert.PanicsWithError(t, "context key not set: nope", func() {
		Get[int](ctx, "nope")
	})
}

func TestGetWrongType(t *testing.T) {
	ctx := withValues(t.Context())
	put(ctx, "verbose", 2)
	assert.PanicsWithError(t, `wrong type: "verbose" is int, not string`, func() {
		Get[string](ctx, "verbose")
	})
}

func TestGetNoBag(t *testing.T) {
	assert.PanicsWithError(t, "context has no values", func() {
		Get[int](t.Context(), "verbose")
	})
}

func TestNilPointerSkipped(t *testing.T) {
	type args struct {
		name *StringArg `ctx:"name"`
	}
	got := ParseOK[args](t)
	ctx := withValues(t.Context())
	bind(ctx, reflect.ValueOf(&got).Elem())
	assert.PanicsWithError(t, "context key not set: name", func() {
		Get[string](ctx, "name")
	})
}
