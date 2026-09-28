package jni

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallRejectsEmptyNames(t *testing.T) {
	_, err := CallStatic("", "dataDir")
	require.ErrorIs(t, err, errEmptyClass)
	_, err = CallStatic("lewkit.Host", "")
	require.ErrorIs(t, err, errEmptyMethod)
	_, err = New(" ")
	require.ErrorIs(t, err, errEmptyClass)
	err = Bind(0, "")
	require.ErrorIs(t, err, errEmptyAnchor)

	var ref *Ref
	_, err = ref.Call("toString")
	require.ErrorIs(t, err, errNilObject)
	ref = &Ref{}
	_, err = ref.Call("toString")
	require.ErrorIs(t, err, errNilObject)
	ref.Release()
}
