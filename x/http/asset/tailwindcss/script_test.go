package tailwindcss

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Script().Render(t.Context(), &buf))
	require.Contains(t, buf.String(), Path)
}
