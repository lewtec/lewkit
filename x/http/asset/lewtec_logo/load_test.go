package lewtec_logo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, Load().Render(t.Context(), &buf))
	require.Contains(t, buf.String(), Path)
	require.NotNil(t, Image())
}
