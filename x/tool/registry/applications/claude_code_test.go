package applications

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaudeCodeListVersionsResolvesChannelsToConcreteVersions(t *testing.T) {
	t.Parallel()

	installed := &claudeCodeTool{
		baseURL: "https://downloads.claude.ai/claude-code-releases",
		fetchURL: func(_ context.Context, url string) ([]byte, error) {
			switch url {
			case "https://downloads.claude.ai/claude-code-releases/latest":
				return []byte("2.1.162\n"), nil
			case "https://downloads.claude.ai/claude-code-releases/stable":
				return []byte("2.1.152\n"), nil
			default:
				return nil, fmt.Errorf("unexpected url %q", url)
			}
		},
	}

	got, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"2.1.162", "2.1.152"}, got)
}

func TestClaudeCodeListArtifactsUsesManifestPlatformBinary(t *testing.T) {
	t.Parallel()

	var installed *claudeCodeTool
	installed = &claudeCodeTool{
		baseURL: "https://downloads.claude.ai/claude-code-releases",
		fetchURL: func(_ context.Context, url string) ([]byte, error) {
			if url != "https://downloads.claude.ai/claude-code-releases/2.1.89/manifest.json" {
				return nil, fmt.Errorf("unexpected url %q", url)
			}

			platform := installed.currentPlatform()
			binary := "claude"
			if strings.HasPrefix(platform, "win32") {
				binary = "claude.exe"
			}

			return []byte(fmt.Sprintf(`{
  "version": "2.1.89",
  "platforms": {
    "%s": {
      "binary": "%s",
      "checksum": "903cb3c96b314d86856632c8702f5cdf971b804d0b19ef87446573bcd1d7df1c",
      "size": 228473472
    }
  }
}`, platform, binary)), nil
		},
	}

	artifacts, err := installed.ListArtifacts(t.Context(), "2.1.89")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	platform := installed.currentPlatform()
	binary := "claude"
	if strings.HasPrefix(platform, "win32") {
		binary = "claude.exe"
	}
	wantURL := fmt.Sprintf("https://downloads.claude.ai/claude-code-releases/2.1.89/%s/%s", platform, binary)
	require.Equal(t, wantURL, artifacts[0].URL)
	require.Equal(t, "sha256:903cb3c96b314d86856632c8702f5cdf971b804d0b19ef87446573bcd1d7df1c", artifacts[0].Hash)
	require.Equal(t, int64(228473472), artifacts[0].Size)
}
