package header

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportName(t *testing.T) {
	require.Equal(t, "OpenFile", ExportName("open_file"))
	require.Equal(t, "Anon", ExportName(""))
	require.Equal(t, "N1st", ExportName("1st"))
	require.Equal(t, "Type", ExportName("type"))
	require.Equal(t, "Name", ExportName("---"))
}
