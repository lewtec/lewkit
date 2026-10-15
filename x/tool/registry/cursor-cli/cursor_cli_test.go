package pkg

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"

	"github.com/stretchr/testify/require"
)

func TestCursorCLIRegistered(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("cursor-cli")
	require.NoError(t, err)

	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	checks := checker.InstallChecks()
	require.Len(t, checks, 2)
	require.Equal(t, "binary:cursor-agent", checks[0].Name())
	require.Equal(t, "binary:agent", checks[1].Name())

	_, isFixer := installed.(tool.Fixer)
	require.True(t, isFixer)
	_, isArtifactTool := installed.(tool.ArtifactTool)
	require.True(t, isArtifactTool)
	_, isBinaryTool := installed.(tool.BinaryTool)
	require.True(t, isBinaryTool)
	_, isPinner := installed.(tool.Pinner)
	require.True(t, isPinner)
}

func TestCursorCLIListVersions(t *testing.T) {
	t.Parallel()

	installed := &cursorCLITool{
		goos:   "linux",
		goarch: "amd64",
		fetchURL: func(_ context.Context, rawURL string) ([]byte, error) {
			require.Equal(t, cursorInstallScriptURL, rawURL)
			return []byte(sampleInstallScript), nil
		},
	}

	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"2026.10.14-941492e", "2026.09.01-abcdef0"}, versions)
}

func TestCursorCLIListVersionsEmpty(t *testing.T) {
	t.Parallel()

	installed := &cursorCLITool{
		goos:   "linux",
		goarch: "amd64",
		fetchURL: func(context.Context, string) ([]byte, error) {
			return []byte("#!/usr/bin/env bash\necho no download url\n"), nil
		},
	}

	_, err := installed.ListVersions(t.Context())
	require.ErrorIs(t, err, ErrNoVersions)
	require.ErrorIs(t, err, install.ErrNoVersions)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestCursorCLIListArtifactsByPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos, goarch string
		wantURL      string
	}{
		{
			goos:    "linux",
			goarch:  "amd64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/linux/x64/agent-cli-package.tar.gz",
		},
		{
			goos:    "linux",
			goarch:  "arm64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/linux/arm64/agent-cli-package.tar.gz",
		},
		{
			goos:    "android",
			goarch:  "arm64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/linux/arm64/agent-cli-package.tar.gz",
		},
		{
			goos:    "darwin",
			goarch:  "amd64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/darwin/x64/agent-cli-package.tar.gz",
		},
		{
			goos:    "darwin",
			goarch:  "arm64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/darwin/arm64/agent-cli-package.tar.gz",
		},
		{
			goos:    "windows",
			goarch:  "amd64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/windows/x64/agent-cli-package.zip",
		},
		{
			goos:    "windows",
			goarch:  "arm64",
			wantURL: "https://downloads.cursor.com/lab/2026.10.14-941492e/windows/arm64/agent-cli-package.zip",
		},
	}

	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			t.Parallel()

			installed := &cursorCLITool{
				goos:   tc.goos,
				goarch: tc.goarch,
				fetchURL: func(_ context.Context, rawURL string) ([]byte, error) {
					if rawURL != cursorInstallScriptURL {
						return nil, fmt.Errorf("%w: %s", errs.ErrNotFound, rawURL)
					}
					return []byte(sampleInstallScript), nil
				},
			}

			artifacts, err := installed.ListArtifacts(t.Context(), "latest")
			require.NoError(t, err)
			require.Len(t, artifacts, 1)
			require.Equal(t, tc.goos, artifacts[0].OS)
			require.Equal(t, tc.goarch, artifacts[0].Arch)
			require.Equal(t, tc.wantURL, artifacts[0].URL)

			pinned, err := installed.ListArtifacts(t.Context(), "v2026.10.14-941492e")
			require.NoError(t, err)
			require.Equal(t, artifacts, pinned)
		})
	}
}

func TestCursorCLIUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		goos, goarch string
	}{
		{"plan9", "amd64"},
		{"linux", "386"},
	} {
		installed := &cursorCLITool{goos: tc.goos, goarch: tc.goarch}
		_, err := installed.ListVersions(t.Context())
		require.ErrorIs(t, err, ErrNoPlatformArtifact)
		require.ErrorIs(t, err, install.ErrNoPlatformArtifact)
		require.ErrorIs(t, err, errs.ErrNotFound)

		_, err = installed.ListArtifacts(t.Context(), "2026.10.14-941492e")
		require.ErrorIs(t, err, ErrNoPlatformArtifact)
		require.ErrorIs(t, err, errs.ErrNotFound)
	}
}

func TestCursorCLIInstallUnixArchiveLinksAgent(t *testing.T) {
	t.Parallel()

	archiveBytes := buildTarGzArchive(t, map[string]string{
		"dist-package/cursor-agent": "#!/usr/bin/env sh\necho 2026.10.14-941492e\n",
		"dist-package/index.js":     "console.log('cursor-agent');\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/install":
			_, _ = writer.Write([]byte(sampleInstallScript))
		case "/lab/2026.10.14-941492e/linux/x64/agent-cli-package.tar.gz":
			_, _ = writer.Write(archiveBytes)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	installed := &cursorCLITool{
		installScriptURL: server.URL + "/install",
		downloadBaseURL:  server.URL + "/lab",
		goos:             "linux",
		goarch:           "amd64",
	}

	destination := t.TempDir()
	cursorAgentPath, err := installed.EnsureBinary(t.Context(), "latest", "cursor-agent", destination)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(destination, "cursor-agent"), cursorAgentPath)

	agentPath := tool.FindBinary(destination, "agent")
	require.Equal(t, filepath.Join(destination, "agent"), agentPath)

	require.NoError(t, tool.RunChecks(t.Context(), destination, installed))
	require.NoError(t, installed.Fix(t.Context(), destination))
}

func TestCursorCLIInstallWindowsArchiveCopiesAgentWrappers(t *testing.T) {
	t.Parallel()

	archiveBytes := buildZipArchive(t, map[string]string{
		"dist-package/cursor-agent.cmd": "@echo off\r\n",
		"dist-package/cursor-agent.ps1": "Write-Output '2026.10.14-941492e'\r\n",
	})

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(archiveBytes)
	}))
	t.Cleanup(server.Close)

	installed := &cursorCLITool{
		downloadBaseURL: server.URL,
		goos:            "windows",
		goarch:          "amd64",
	}

	destination := t.TempDir()
	err := installed.InstallArtifact(t.Context(), tool.Artifact{
		OS:   "windows",
		Arch: "amd64",
		URL:  server.URL + "/2026.10.14-941492e/windows/x64/agent-cli-package.zip",
	}, destination)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(destination, "cursor-agent.cmd"), tool.FindBinary(destination, "cursor-agent"))
	require.Equal(t, filepath.Join(destination, "agent.cmd"), tool.FindBinary(destination, "agent"))

	ps1Body, err := os.ReadFile(filepath.Join(destination, "agent.ps1"))
	require.NoError(t, err)
	require.Equal(t, "Write-Output '2026.10.14-941492e'\r\n", string(ps1Body))
}

func buildTarGzArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		require.NoError(t, tarWriter.WriteHeader(header))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return buffer.Bytes()
}

func buildZipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	for name, content := range files {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		header.SetMode(0o755)
		writer, err := zipWriter.CreateHeader(header)
		require.NoError(t, err)
		_, err = writer.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, zipWriter.Close())
	return buffer.Bytes()
}

const sampleInstallScript = `#!/usr/bin/env bash
set -euo pipefail
DOWNLOAD_URL="https://downloads.cursor.com/lab/2026.10.14-941492e/${OS}/${ARCH}/agent-cli-package.tar.gz"
PREVIOUS_URL="https://downloads.cursor.com/lab/2026.09.01-abcdef0/${OS}/${ARCH}/agent-cli-package.tar.gz"
DUPLICATE_URL="https://downloads.cursor.com/lab/2026.10.14-941492e/${OS}/${ARCH}/agent-cli-package.tar.gz"
`
