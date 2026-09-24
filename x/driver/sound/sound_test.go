package sound

import (
	"testing"

	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestOpenRejectsFormat(t *testing.T) {
	_, err := Open(t.Context(), Config{})
	require.ErrorIs(t, err, pcs.ErrFormat)
}
