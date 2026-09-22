package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrokPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "linux-x86_64"},
		{"linux", "arm64", "linux-aarch64"},
		{"darwin", "amd64", "macos-x86_64"},
		{"darwin", "arm64", "macos-aarch64"},
		{"windows", "amd64", "windows-x86_64"},
		{"windows", "arm64", "windows-aarch64"},
		{"android", "arm64", "linux-aarch64"},
		{"android", "amd64", "linux-x86_64"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, grokPlatform(tc.goos, tc.goarch))
	}
}
