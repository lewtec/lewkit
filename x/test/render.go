package test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// RenderContains renders with t.Context and requires needle in the text.
func RenderContains(t *testing.T, render func(context.Context, io.Writer) error, needle string) {
	t.Helper()
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, render(t.Context(), &buf))
	require.Contains(t, buf.String(), needle)
}
