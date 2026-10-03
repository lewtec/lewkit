package jni

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func TestAsRef(t *testing.T) {
	got, err := AsRef(nil, nil)
	require.NoError(t, err)
	require.Nil(t, got)

	_, err = AsRef(nil, errBoom)
	require.EqualError(t, err, "boom")

	_, err = AsRef("nope", nil)
	require.ErrorIs(t, err, errJavaValue)

	ref := &Ref{}
	got, err = AsRef(ref, nil)
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

	wide, err := Float(float32(0.5), nil)
	require.NoError(t, err)
	require.InDelta(t, 0.5, wide, 0)
	exact, err := Float(0.25, nil)
	require.NoError(t, err)
	require.Equal(t, 0.25, exact)
	_, err = Float(1, nil)
	require.ErrorIs(t, err, errJavaValue)
	_, err = Float(0.5, errBoom)
	require.EqualError(t, err, "boom")
}
