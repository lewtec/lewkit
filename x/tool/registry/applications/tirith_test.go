package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTirithListVersionsSkipsThreatDatabaseReleases(t *testing.T) {
	t.Parallel()

	installed := tirithTool{inner: stubTool{versions: []string{
		"threatdb-26940486720-1",
		"threatdb-26874685865-1",
		"v0.3.1",
		"threatdb-25594072496-1",
		"v0.3.0",
		"v0.2.12",
	}}}

	got, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"v0.3.1", "v0.3.0", "v0.2.12"}, got)
}
