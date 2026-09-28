package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveName(t *testing.T) {
	require.Equal(t, "demo_Linux_x86_64.tar.gz", ArchiveName("demo", Target{GOOS: "linux", GOARCH: "amd64"}))
	require.Equal(t, "demo_Windows_arm64.zip", ArchiveName("demo", Target{GOOS: "windows", GOARCH: "arm64"}))
	require.Len(t, DesktopTargets(), 6)
}
