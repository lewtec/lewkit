package mise

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToolRejectsEmptyRef(t *testing.T) {
	_, err := NewTool("  ")
	require.ErrorIs(t, err, ErrEmptyMiseRef)
}
