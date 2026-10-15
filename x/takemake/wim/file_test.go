package wim

import (
	"bytes"
	"io"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestScaffold(t *testing.T) {
	t.Parallel()
	file := New()
	require.NotNil(t, file)
	_, err := Take(bytes.NewReader(nil))
	require.ErrorIs(t, err, errs.ErrScaffold)
	require.ErrorIs(t, file.Make(io.Discard), errs.ErrScaffold)
}
