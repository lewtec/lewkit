package nativeerr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver/present"
)

func TestMap(t *testing.T) {
	lost := errors.New("lost")
	size := errors.New("size")
	closed := errors.New("closed")
	other := errors.New("other")
	require.NoError(t, Map(nil, lost, size, closed))
	require.ErrorIs(t, Map(fmt.Errorf("%w", lost), lost, size, closed), present.ErrLost)
	require.ErrorIs(t, Map(size, lost, size, closed), present.ErrSize)
	require.ErrorIs(t, Map(closed, lost, size, closed), present.ErrClosed)
	require.ErrorIs(t, Map(other, lost, size, closed), other)
}
