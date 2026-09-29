package android

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/jni"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func TestRef(t *testing.T) {
	got, err := Ref(nil, nil)
	require.NoError(t, err)
	require.Nil(t, got)

	_, err = Ref(nil, errBoom)
	require.EqualError(t, err, "boom")

	_, err = Ref("nope", nil)
	require.ErrorIs(t, err, errJavaValue)

	ref := &jni.Ref{}
	got, err = Ref(ref, nil)
	require.NoError(t, err)
	require.Same(t, ref, got)
}

func TestPlainValues(t *testing.T) {
	n, err := Int(7, nil)
	require.NoError(t, err)
	require.Equal(t, 7, n)
	_, err = Int("7", nil)
	require.ErrorIs(t, err, errJavaValue)

	text, err := Text("files", nil)
	require.NoError(t, err)
	require.Equal(t, "files", text)
	_, err = Text(1, nil)
	require.ErrorIs(t, err, errJavaValue)

	on, err := Bool(true, nil)
	require.NoError(t, err)
	require.True(t, on)
	_, err = Bool(1, nil)
	require.ErrorIs(t, err, errJavaValue)
}
