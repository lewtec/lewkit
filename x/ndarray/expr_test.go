package ndarray

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadyChecksNilBeforeError(t *testing.T) {
	bad := failed(ErrType)
	ok := &node{}

	got := ready(nil)
	require.ErrorIs(t, got.err, ErrOp)

	got = ready(bad, nil)
	require.ErrorIs(t, got.err, ErrOp)

	got = ready(nil, bad)
	require.ErrorIs(t, got.err, ErrOp)

	got = ready(bad, ok)
	require.Same(t, bad, got)

	got = ready(ok, bad)
	require.Same(t, bad, got)

	require.Nil(t, ready(ok, ok))
	require.Nil(t, ready())
}
