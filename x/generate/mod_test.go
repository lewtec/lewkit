package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleLine(t *testing.T) {
	t.Parallel()
	got, err := ModuleLine([]byte("module example.com/app\n\ngo 1.27\n"))
	require.NoError(t, err)
	assert.Equal(t, "example.com/app", got)

	_, err = ModuleLine([]byte("go 1.27\n"))
	assert.ErrorIs(t, err, ErrNoModule)
}
