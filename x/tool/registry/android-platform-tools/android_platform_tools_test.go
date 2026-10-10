package pkg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	_ "github.com/lewtec/lewkit/x/driver/fetchurl/native"
	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"

	"github.com/stretchr/testify/require"
)

var errUnexpectedTestURL = errors.New("unexpected test url")

func TestAndroidPlatformToolsRegistered(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("android-platform-tools")
	require.NoError(t, err)
	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	var names []string
	for _, check := range checker.InstallChecks() {
		names = append(names, check.Name())
	}
	require.Equal(t, []string{
		"binary:adb",
		"binary:fastboot",
		"binary:etc1tool",
		"binary:hprof-conv",
		"binary:make_f2fs",
		"binary:make_f2fs_casefold",
		"binary:mke2fs",
		"binary:sqlite3",
	}, names)
}

func TestPlatformToolsBuildsSelectHostArchive(t *testing.T) {
	t.Parallel()

	body := platformToolsIndexFixture()
	cases := []struct {
		name    string
		goos    string
		goarch  string
		version string
		urlPart string
		hash    string
		size    int64
	}{
		{
			name:    "linux amd64 keeps a generic zip when no arch is named",
			goos:    "linux",
			goarch:  "amd64",
			version: "36.0.0",
			urlPart: "/platform-tools_r36.0.0-linux.zip",
			hash:    "sha1:" + strings.Repeat("a", 40),
			size:    10,
		},
		{
			name:    "linux amd64 prefers the zip that names x86_64",
			goos:    "linux",
			goarch:  "amd64",
			version: "37.0.1",
			urlPart: "/platform-tools_r37.0.1-linux-x86_64.zip",
			hash:    "sha1:" + strings.Repeat("f", 40),
			size:    23,
		},
		{
			name:    "linux arm64 prefers the archive that names the arch",
			goos:    "linux",
			goarch:  "arm64",
			version: "37.0.1",
			urlPart: "https://example.test/platform-tools_r37.0.1-linux-aarch64.zip",
			hash:    "sha256:" + strings.Repeat("e", 64),
			size:    21,
		},
		{
			name:    "darwin arm64 uses the universal zip",
			goos:    "darwin",
			goarch:  "arm64",
			version: "36.0.0",
			urlPart: "/platform-tools_r36.0.0-darwin.zip",
			hash:    "sha1:" + strings.Repeat("b", 40),
			size:    11,
		},
		{
			name:    "darwin amd64 uses the same universal zip",
			goos:    "darwin",
			goarch:  "amd64",
			version: "36.0.0",
			urlPart: "/platform-tools_r36.0.0-darwin.zip",
			hash:    "sha1:" + strings.Repeat("b", 40),
			size:    11,
		},
		{
			name:    "windows amd64",
			goos:    "windows",
			goarch:  "amd64",
			version: "36.0.0",
			urlPart: "/platform-tools_r36.0.0-win.zip",
			hash:    "sha1:" + strings.Repeat("c", 40),
			size:    12,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builds, err := platformToolsBuilds([]byte(body), tt.goos, tt.goarch)
			require.NoError(t, err)
			var got platformToolsBuild
			found := false
			for _, build := range builds {
				if build.Version == tt.version {
					got = build
					found = true
					break
				}
			}
			require.True(t, found, "version %s for %s/%s", tt.version, tt.goos, tt.goarch)
			require.Contains(t, got.URL, tt.urlPart)
			require.Equal(t, tt.hash, got.Hash)
			require.Equal(t, tt.size, got.Size)
			require.Equal(t, tt.goarch, got.Arch)
		})
	}
}

func TestPlatformToolsBuildsSkipOtherArch(t *testing.T) {
	t.Parallel()

	body := []byte(platformToolsIndexFixture())
	builds, err := platformToolsBuilds(body, "linux", "386")
	require.NoError(t, err)
	require.Empty(t, builds)

	builds, err = platformToolsBuilds(body, "windows", "arm64")
	require.NoError(t, err)
	require.Empty(t, builds)
}

func TestPlatformToolsListVersionsOrdersNewestFirst(t *testing.T) {
	t.Parallel()

	installed := platformToolsFixtureTool()
	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"37.0.1", "36.0.0"}, versions)

	artifacts, err := installed.ListArtifacts(t.Context(), "r37.0.1")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, artifacts[0].URL, "platform-tools_r37.0.1-linux-x86_64.zip")
	require.Equal(t, "sha1:"+strings.Repeat("f", 40), artifacts[0].Hash)
	require.Equal(t, "linux", artifacts[0].OS)
	require.Equal(t, "amd64", artifacts[0].Arch)

	artifacts, err = installed.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)
	require.Contains(t, artifacts[0].URL, "platform-tools_r37.0.1-linux-x86_64.zip")

	_, err = installed.ListArtifacts(t.Context(), "1.0.0")
	require.ErrorIs(t, err, install.ErrNoPlatformArtifact)
}

func TestPlatformToolsIndexRejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := platformToolsBuilds([]byte("<sdk-repository>"), "linux", "amd64")
	require.Error(t, err)
}

func TestAndroidPlatformToolsLiveIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("live index")
	}
	t.Parallel()

	raw, err := newAndroidPlatformTools()
	require.NoError(t, err)
	installed := raw.(*androidPlatformToolsTool)
	ctx := t.Context()

	versions, err := installed.ListVersions(ctx)
	if errors.Is(err, install.ErrNoVersions) {
		t.Skip("no platform-tools archive for " + runtime.GOOS + "/" + runtime.GOARCH)
	}
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	artifacts, err := installed.ListArtifacts(ctx, versions[0])
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, artifacts[0].URL, "https://dl.google.com/android/repository/platform-tools_r"+versions[0])
	require.Contains(t, artifacts[0].Hash, "sha1:")
	require.Equal(t, runtime.GOOS, artifacts[0].OS)
	require.Equal(t, runtime.GOARCH, artifacts[0].Arch)
	require.Positive(t, artifacts[0].Size)

	prefixed, err := installed.ListArtifacts(ctx, "v"+versions[0])
	require.NoError(t, err)
	require.Equal(t, artifacts[0].URL, prefixed[0].URL)
}

func TestAndroidPlatformToolsInstall(t *testing.T) {
	if os.Getenv("MODOT_TEST_TOOL_INSTALL") != "1" {
		t.Skip("set MODOT_TEST_TOOL_INSTALL=1 to download platform-tools")
	}

	installed, err := newAndroidPlatformTools()
	require.NoError(t, err)
	ctx := t.Context()
	if _, err := installed.ListVersions(ctx); errors.Is(err, install.ErrNoVersions) {
		t.Skip("no platform-tools archive for " + runtime.GOOS + "/" + runtime.GOARCH)
	}
	dest := t.TempDir()
	require.NoError(t, installed.Install(ctx, "latest", dest))
	require.NoError(t, tool.RunChecks(ctx, dest, installed))

	for _, name := range []string{"adb", "fastboot", "sqlite3"} {
		bin := tool.FindBinary(dest, name)
		require.NotEmpty(t, bin, name)
		args := []string{"version"}
		if name == "sqlite3" {
			args = []string{"-version"}
		}
		if name == "fastboot" {
			args = []string{"--version"}
		}
		out, err := execdriver.OutputString(ctx, bin, args...)
		require.NoError(t, err, name)
		require.NotEmpty(t, strings.TrimSpace(out), name)
	}
}

func platformToolsFixtureTool() *androidPlatformToolsTool {
	return &androidPlatformToolsTool{
		indexURL: androidPlatformToolsIndexURL,
		goos:     "linux",
		goarch:   "amd64",
		fetchURL: func(_ context.Context, gotURL string) ([]byte, error) {
			if gotURL != androidPlatformToolsIndexURL {
				return nil, fmt.Errorf("unexpected url %q: %w", gotURL, errUnexpectedTestURL)
			}
			return []byte(platformToolsIndexFixture()), nil
		},
	}
}

func platformToolsIndexFixture() string {
	sha1a := strings.Repeat("a", 40)
	sha1b := strings.Repeat("b", 40)
	sha1c := strings.Repeat("c", 40)
	sha1d := strings.Repeat("d", 40)
	sha256e := strings.Repeat("e", 64)
	sha1f := strings.Repeat("f", 40)
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<sdk:sdk-repository xmlns:sdk="http://schemas.android.com/sdk/android/repo/repository2/01" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <license id="android-sdk-license" type="text">ignored</license>
  <remotePackage path="tools">
    <revision><major>1</major><minor>0</minor><micro>0</micro></revision>
  </remotePackage>
  <remotePackage path="platform-tools">
    <revision><major>36</major><minor>0</minor><micro>0</micro></revision>
    <archives>
      <archive>
        <complete>
          <size>10</size>
          <checksum>%s</checksum>
          <url>platform-tools_r36.0.0-linux.zip</url>
        </complete>
        <host-os>linux</host-os>
      </archive>
      <archive>
        <complete>
          <size>11</size>
          <checksum type="sha1">%s</checksum>
          <url>platform-tools_r36.0.0-darwin.zip</url>
        </complete>
        <host-os>macosx</host-os>
      </archive>
      <archive>
        <complete>
          <size>12</size>
          <checksum>%s</checksum>
          <url>platform-tools_r36.0.0-win.zip</url>
        </complete>
        <host-os>windows</host-os>
      </archive>
    </archives>
  </remotePackage>
  <remotePackage path="platform-tools">
    <revision><major>37</major><minor>0</minor><micro>1</micro></revision>
    <archives>
      <archive>
        <complete>
          <size>20</size>
          <checksum>%s</checksum>
          <url>platform-tools_r37.0.1-linux.zip</url>
        </complete>
        <host-os>linux</host-os>
      </archive>
      <archive>
        <complete>
          <size>23</size>
          <checksum>%s</checksum>
          <url>platform-tools_r37.0.1-linux-x86_64.zip</url>
        </complete>
        <host-os>linux</host-os>
      </archive>
      <archive>
        <complete>
          <size>21</size>
          <checksum type="sha256">%s</checksum>
          <url>https://example.test/platform-tools_r37.0.1-linux-aarch64.zip</url>
        </complete>
        <host-os>linux</host-os>
        <host-arch>aarch64</host-arch>
      </archive>
      <archive>
        <complete>
          <size>22</size>
          <checksum>short</checksum>
          <url>platform-tools_r37.0.1-linux-unhashed.zip</url>
        </complete>
        <host-os>linux</host-os>
      </archive>
    </archives>
  </remotePackage>
</sdk:sdk-repository>`, sha1a, sha1b, sha1c, sha1d, sha1f, sha256e)
}
