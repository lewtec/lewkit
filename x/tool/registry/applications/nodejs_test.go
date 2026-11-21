package applications

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodejsListArtifactsAcceptsVersionWithoutVPrefix(t *testing.T) {
	t.Parallel()

	installed := &nodejsTool{}
	artifacts, err := installed.ListArtifacts(t.Context(), "22.16.0")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	osPart, archPart, ext := installed.nodePlatformAndExt()
	wantFilename := fmt.Sprintf("node-v22.16.0-%s-%s%s", osPart, archPart, ext)
	wantURL := fmt.Sprintf("https://nodejs.org/dist/v22.16.0/%s", wantFilename)
	require.Equal(t, wantURL, artifacts[0].URL)
	if runtime.GOOS != "windows" {
		require.Contains(t, artifacts[0].URL, ".tar.gz")
	}
}
