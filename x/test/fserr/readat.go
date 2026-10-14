// Package fserr checks filesystem open errors shared by the archive readers.
package fserr

import (
	"errors"
	"io/fs"
	"testing"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NeedReadAt checks that err is a path error from open with [lewfs.ErrNeedReadAt].
func NeedReadAt(tb testing.TB, err error) {
	tb.Helper()
	require.ErrorIs(tb, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(tb, ok)
	assert.Equal(tb, "open", pe.Op)
}
