package sakuracss

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStylesheet(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Stylesheet().Render(t.Context(), &buf))
	require.Contains(t, buf.String(), Path)
}
