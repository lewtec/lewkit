package text

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDigits(t *testing.T) {
	require.False(t, Digits(""))
	require.False(t, Digits("12a"))
	require.False(t, Digits("١"))
	require.True(t, Digits("0"))
	require.True(t, Digits("0042"))
}
