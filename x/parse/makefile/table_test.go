package makefile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTableMatchesDeclaration(t *testing.T) {
	table, err := ParseTable()
	require.NoError(t, err)
	require.Equal(t, table, []byte(makefileTable))
}
