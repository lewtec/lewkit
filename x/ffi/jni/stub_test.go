//go:build !android || !cgo

package jni

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStubIsUnavailable(t *testing.T) {
	_, err := CallStatic("lewkit.Host", "dataDir")
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = New("java.lang.Object")
	require.ErrorIs(t, err, ErrUnavailable)
	err = Bind(0, "lewkit/Host")
	require.ErrorIs(t, err, ErrUnavailable)
	ref := &Ref{ptr: 1, class: "java.lang.Object"}
	_, err = ref.Call("toString")
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = Class("java.lang.Object")
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = StaticField("lewkit.Host", "app")
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = ref.Field("uiMode")
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = Proxy("java.lang.Runnable", func(string, []any) (any, error) { return nil, nil })
	require.ErrorIs(t, err, ErrUnavailable)
}
