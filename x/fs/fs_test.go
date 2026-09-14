package fs

import (
	"errors"
	"io"
	iofs "io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

func TestReaderAt(t *testing.T) {
	t.Parallel()
	_, err := ReaderAt("open", onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, ErrNeedReadAt)
	pe, ok := errors.AsType[*iofs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	ra, err := ReaderAt("open", strings.NewReader("x"))
	require.NoError(t, err)
	assert.NotNil(t, ra)
}
