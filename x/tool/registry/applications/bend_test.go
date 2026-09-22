package applications

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

var errUnexpectedTestURL = errors.New("unexpected test url")

func TestValidBendVersion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{"2.0.5", true},
		{"2.0.0", true},
		{"v2.0.5", false},
		{"", false},
		{".", false},
		{"..", false},
		{"../evil", false},
		{"2.0.5/../x", false},
		{"2.0.5.tar.gz", false},
		{"2_0-5", false},
		{"2..0.5", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, validBendVersion(tc.in))
		})
	}
}

func TestBendListVersionsReadsLatestJSON(t *testing.T) {
	t.Parallel()
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)

	got, err := tool.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"2.0.5"}, got)
}

func TestBendListArtifactsUsesLatestHash(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	const sha = "4db70e77ce1b1027f1d0e15dee025921fa794a9b415add4350ec7c64acf2775b"
	tool := newTestBend(t, fmt.Sprintf(`{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":%q}`, sha))

	arts, err := tool.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "https://bend-lang.com/dl/2.0.5.tar.gz", arts[0].URL)
	require.Equal(t, "sha256:"+sha, arts[0].Hash)
}

func TestBendListArtifactsPinnedVersionOmitsUnknownHash(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)

	arts, err := tool.ListArtifacts(t.Context(), "2.0.4")
	require.NoError(t, err)
	require.Len(t, arts, 1)
	require.Equal(t, "https://example.test/dl/2.0.4.tar.gz", arts[0].URL)
	require.Empty(t, arts[0].Hash)
}

func TestBendListArtifactsRejectsUnsafeVersion(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("bend is not packaged for windows")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)
	_, err := tool.ListArtifacts(t.Context(), "../evil")
	require.ErrorIs(t, err, ErrInvalidBendVersion)
}

func TestBendListArtifactsWindowsUnsupported(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("windows-only assertion")
	}
	tool := newTestBend(t, `{"ver":"2.0.5","url":"https://bend-lang.com/dl/2.0.5.tar.gz","sha256":"abcd"}`)
	_, err := tool.ListArtifacts(t.Context(), "2.0.5")
	require.ErrorIs(t, err, ErrNoPlatformArtifact)
}

func TestWriteBendLauncher(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	require.NoError(t, writeBendLauncher(dest))
	path := filepath.Join(dest, "bin", "bend")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	script := string(got)
	for _, want := range []string{
		"#!/bin/sh",
		"BEND_NO_TELEMETRY=1",
		`ws=$(command -v workspaced)`,
		`exec -a bun "$ws" tool with bun -- bun "$main" "$@"`,
		"bend2/main.ts",
	} {
		require.Contains(t, script, want)
	}
	require.NotContains(t, script, "XDG_DATA_HOME")
	require.NotContains(t, script, ".local/share")
	require.NotContains(t, script, "/tmp/go-build")
	require.NotContains(t, script, "curl")
	require.NotContains(t, script, "bend-lang.com/ping")
}

func TestBendEnrichLockfile(t *testing.T) {
	t.Parallel()
	pin := (&bendTool{}).Pin()
	require.Equal(t, "semver", pin.Versioning)
}

func newTestBend(t *testing.T, latestJSON string) *bendTool {
	t.Helper()
	const origin = "https://example.test"
	return &bendTool{
		origin: origin,
		fetchURL: func(_ context.Context, u string) ([]byte, error) {
			if u != origin+"/dl/latest.json" {
				return nil, fmt.Errorf("unexpected url %q: %w", u, errUnexpectedTestURL)
			}
			return []byte(latestJSON), nil
		},
	}
}
