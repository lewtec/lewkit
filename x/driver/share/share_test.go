package share

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate_Empty(t *testing.T) {
	err := Item{}.validate()
	require.ErrorIs(t, err, ErrEmptyItem)
}

func TestValidate_RelativePath(t *testing.T) {
	err := Item{Paths: []string{"rel"}}.validate()
	require.Error(t, err, "expected error")
}

func TestValidate_Text(t *testing.T) {
	require.NoError(t, (Item{Text: "hi"}).validate())
}
